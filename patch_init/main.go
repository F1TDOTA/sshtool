package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"patch_init/pkg"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	_ "github.com/browserutils/kooky/browser/all" // register cookie store finders!
	"github.com/mozillazg/go-pinyin"
)

var strUrl = "http://192.168.46.152:8081"

type BugInfo struct {
	bugId string
}

// 从chrome cookie文件中获取http://192.168.46.152:8081/的cookie，并返回
func getCookieFromChrom(strAddr string) string {
	return ""
}

// 传入id,使用GET方法访问http://192.168.46.152:8081/www/index.php?m=bug&f=view&bugID=61460获取对应的信息，并进行解析，获取ID，客户名称
func get_bug_info(strBugId string) {

}

func build_local_dir() {

}

func check_local_code() {

}

func newBugBranch() {

}

func build_ssh_dir() {

}

func check_compile_code() {

}

func EnsureDirExists(path string) error {
	// 判断是否存在
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		// 不存在则创建（含父目录）
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return fmt.Errorf("创建目录失败: %v", err)
		}
		fmt.Println("已创建目录:", path)
		return nil
	}

	// 存在但不是目录
	if !info.IsDir() {
		return fmt.Errorf("%s 存在但不是目录", path)
	}

	fmt.Println("目录已存在:", path)
	return nil
}

func fetchWithCookies(url string, cookieHeader string) error {

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Cookie", cookieHeader)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/141.0.0.0 Safari/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// 匹配编号
	reID := regexp.MustCompile(`copyContent\('(\d+)'\)`)
	idMatches := reID.FindStringSubmatch(bodyStr)
	id := ""
	if len(idMatches) > 1 {
		id = idMatches[1]
	}

	// 匹配客户单位
	reCustomer := regexp.MustCompile(`<th>\s*客户单位\s*</th>\s*<td>([^<]+)</td>`)
	customerMatches := reCustomer.FindStringSubmatch(bodyStr)
	customer := ""
	if len(customerMatches) > 1 {
		customer = customerMatches[1]
	}

	//引擎版本
	reEngVer := regexp.MustCompile(`<th>\s*引擎版本\s*</th>\s*<td>([^<]+)</td>`)
	engineVerMatches := reEngVer.FindStringSubmatch(bodyStr)
	engineVer := ""
	if len(engineVerMatches) > 1 {
		engineVer = engineVerMatches[1]
	}

	// 获取客户单位拼音
	pyArgs := pinyin.NewArgs()
	pyArgs.Style = pinyin.Normal
	py := pinyin.Pinyin(customer, pyArgs)

	var builder strings.Builder
	for _, item := range py {
		if len(item) > 0 {
			r := []rune(item[0])
			if len(r) > 0 {
				builder.WriteRune(unicode.ToUpper(r[0]))
			}
		}
	}
	strCustomerPinYin := builder.String()
	fmt.Printf("\n编号: %s\n客户单位: %s\n引擎版本: %s\n客户单位拼音: %s\n", id, customer, engineVer, strCustomerPinYin)

	// 新建目录
	strProjectRoot := "F:\\dk_code"
	strProjectName := fmt.Sprintf("%s_%s_%s_%s", strCustomerPinYin, id, customer, engineVer)
	strProjectDir := filepath.Join(strProjectRoot, strProjectName)
	if err := EnsureDirExists(strProjectDir); err != nil {
		fmt.Println("错误:", err)
		return fmt.Errorf("create project dir fail: %v", err)
	}

	// 写ini文件
	file, err := os.Create(filepath.Join(strProjectDir, "info.ini"))
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer file.Close()

	fmt.Fprintf(file, "%s=%s\n", "id", id)
	fmt.Fprintf(file, "%s=%s\n", "cusName", customer)
	fmt.Fprintf(file, "%s=%s\n", "cusPinYinName", strCustomerPinYin)
	fmt.Fprintf(file, "%s=%s\n", "version", engineVer)
	fmt.Fprintf(file, "%s=%s\n", "projectDir", strProjectName)

	return nil

}

func main() {
	//cookiesFile := "C:\\Users\\Administrator\\AppData\\Local\\Google\\Chrome\\User Data\\Default\\Network\\Cookies"
	cookieObj := pkg.NewChromeCookie()
	strCookie, _ := cookieObj.GetHostCookie("192.168.46.152")
	fmt.Printf("%v", strCookie)
	fetchWithCookies("http://192.168.46.152:8081/www/index.php?m=bug&f=view&bugID=61575", strCookie)
}
