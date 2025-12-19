package pkg

import (
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	_ "modernc.org/sqlite"
)

type BrowserCookie struct {
	Host  string
	Name  string
	Value string
}

type ChromeCookie struct {
	strLocalAppData string
	strProfilePath  string
}

func NewChromeCookie() *ChromeCookie {
	return &ChromeCookie{
		strLocalAppData: "",
		strProfilePath:  "",
	}
}

func (c *ChromeCookie) getWinChromeProfilePath() (string, error) {

	candidates := []string{
		filepath.Join(c.strLocalAppData, "Google", "Chrome", "User Data", "Default"),
		filepath.Join(c.strLocalAppData, "Google", "Chrome", "User Data", "Profile 1"),
		filepath.Join(c.strLocalAppData, "Google", "Chrome", "User Data", "Profile 2"),
		filepath.Join(c.strLocalAppData, "Chromium", "User Data", "Default"),
	}

	for _, p := range candidates {
		// 新路径
		strNewProfilePath := filepath.Join(p, "Network")
		strNewCookiePath := filepath.Join(strNewProfilePath, "Cookies")
		if fi, err := os.Stat(strNewCookiePath); err == nil && !fi.IsDir() {
			return strNewProfilePath, nil
		}

		// 老路径
		strOldCookiePath := filepath.Join(p, "Cookies")
		if fi, err := os.Stat(strOldCookiePath); err == nil && !fi.IsDir() {
			return p, nil
		}
	}

	return "", fmt.Errorf("cookie path not found")
}

func (c *ChromeCookie) decryptDPAPI(encrypted []byte) ([]byte, error) {
	if len(encrypted) == 0 {
		return nil, fmt.Errorf("empty input to decryptDPAPI")
	}

	// 构造输入 DataBlob
	var inBlob windows.DataBlob
	inBlob.Data = &encrypted[0]
	inBlob.Size = uint32(len(encrypted))

	var outBlob windows.DataBlob

	// 调用系统 API
	err := windows.CryptUnprotectData(
		&inBlob,
		nil,
		nil,
		0,
		nil,
		0,
		&outBlob,
	)
	if err != nil {
		return nil, fmt.Errorf("CryptUnprotectData failed: %v", err)
	}

	// 从返回的 DataBlob 复制出字节数据
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(outBlob.Data)))
	decrypted := unsafe.Slice(outBlob.Data, outBlob.Size)

	// 拷贝到新切片（避免指针释放后访问）
	result := make([]byte, len(decrypted))
	copy(result, decrypted)

	return result, nil
}

func (c *ChromeCookie) getDecryptAesKey() ([]byte, error) {

	statePath := filepath.Join(c.strLocalAppData, "Google", "Chrome", "User Data", "Local State")
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %s", statePath)
	}

	var state map[string]any
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	osc, ok := state["os_crypt"].(map[string]any)
	if !ok {
		return nil, errors.New("Local State missing os_crypt")
	}

	encKeyB64, ok := osc["encrypted_key"].(string)
	if !ok {
		return nil, errors.New("Local State missing os_crypt.encrypted_key")
	}

	raw, err := base64.StdEncoding.DecodeString(encKeyB64)
	if err != nil {
		return nil, fmt.Errorf("decode encrypted_key: %w", err)
	}

	// Windows 前缀为 "DPAPI"
	const prefix = "DPAPI"
	if !strings.HasPrefix(string(raw), prefix) || len(raw) <= len(prefix) {
		return nil, errors.New("unexpected encrypted_key format")
	}
	dpapiBytes := raw[len(prefix):]

	return c.decryptDPAPI(dpapiBytes)
}

func (c *ChromeCookie) joinCookies(cookies []BrowserCookie) string {
	var cookieStr strings.Builder
	for i, ck := range cookies {
		if i > 0 {
			cookieStr.WriteString("; ")
		}
		cookieStr.WriteString(fmt.Sprintf("%s=%s", ck.Name, ck.Value))
	}
	return cookieStr.String()
}

func (c *ChromeCookie) GetHostCookie(strHost string) (string, error) {
	c.strLocalAppData = os.Getenv("LOCALAPPDATA")
	if c.strLocalAppData == "" {
		return "", fmt.Errorf("Env variable LOCALAPPDATA is not set")
	}

	strProfilePath, err := c.getWinChromeProfilePath()
	if err != nil {
		return "", fmt.Errorf("Failed to get Windows Profile Path: %v", err)
	}
	c.strProfilePath = strProfilePath

	aesMasterKey, err := c.getDecryptAesKey()
	if err != nil {
		return "", fmt.Errorf("Failed to decrypt key: %v", err)
	}

	strCookiePath := filepath.Join(strProfilePath, "Cookies")
	fmt.Printf("%v\n", aesMasterKey)

	cookies, err := c.readAllCookiesFromDB(strCookiePath, strHost)
	if err != nil {
		fmt.Printf("read cookies error: %v\n", err)
		os.Exit(1)
	}

	for _, c := range cookies {
		fmt.Printf("[%s] %s=%s\n", c.Host, c.Name, c.Value)
	}
	fmt.Printf("Total: %d\n", len(cookies))
	cookieHeader := c.joinCookies(cookies)

	return cookieHeader, nil
}

func (c *ChromeCookie) decryptAESGCMChrome(enc []byte, masterKey []byte) (string, error) {
	if len(enc) == 0 {
		return "", nil
	}
	// 明文存放在 value 列；若 value 为空且 encrypted_value 有内容：
	// Windows AES-GCM 前缀通常是 "v10" 或 "v11"
	if len(enc) > 3 && (enc[0] == 'v' && enc[1] == '1' && (enc[2] == '0' || enc[2] == '1')) {
		enc = enc[3:]
	}

	if len(enc) < 12+1 {
		return "", errors.New("invalid encrypted cookie length")
	}
	nonce := enc[:12]
	ciphertext := enc[12:]

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", fmt.Errorf("NewCipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("NewGCM: %w", err)
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("GCM open: %w", err)
	}
	return string(plain), nil
}

// 将 cookies DB 复制到临时文件，避免被 Chrome 锁定
func (c *ChromeCookie) copyToTemp(src string) (string, func(), error) {
	// 使用 Windows API 共享读模式打开，避免锁冲突
	f, err := os.OpenFile(src, os.O_RDONLY, 0)
	if err != nil {
		return "", nil, fmt.Errorf("open source db failed: %w", err)
	}
	defer f.Close()

	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("cookies_copy_%d.sqlite", time.Now().UnixNano()))
	out, err := os.Create(tmp)
	if err != nil {
		return "", nil, fmt.Errorf("create temp db: %w", err)
	}

	if _, err := io.Copy(out, f); err != nil {
		out.Close()
		os.Remove(tmp)
		return "", nil, fmt.Errorf("copy db: %w", err)
	}

	out.Close()
	cleanup := func() { _ = os.Remove(tmp) }
	return tmp, cleanup, nil
}

func (c *ChromeCookie) readAllCookiesFromDB(dbPath string, domainFilter string) ([]BrowserCookie, error) {
	masterKey, err := c.getDecryptAesKey()
	if err != nil {
		return nil, fmt.Errorf("get master key: %w", err)
	}

	tmpDB, cleanup, err := c.copyToTemp(dbPath)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// _query: 如果带域名过滤
	query := `SELECT host_key, name, value, encrypted_value FROM cookies`
	args := []any{}
	if domainFilter != "" {
		query += ` WHERE host_key LIKE ?`
		// 支持 .example.com 和 example.com
		args = append(args, "%"+domainFilter)
	}

	db, err := sql.Open("sqlite", tmpDB)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query cookies: %w", err)
	}
	defer rows.Close()

	var res []BrowserCookie
	for rows.Next() {
		var host, name, value string
		var enc []byte
		if err := rows.Scan(&host, &name, &value, &enc); err != nil {
			continue
		}
		// value 优先；若空则尝试解密 encrypted_value
		if value == "" && len(enc) > 0 {
			if plain, err := c.decryptAESGCMChrome(enc, masterKey); err == nil {
				value = plain
			}
		}
		res = append(res, BrowserCookie{Host: host, Name: name, Value: value})
	}
	return res, nil
}
