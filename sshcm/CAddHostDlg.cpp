// CAddHostDlg.cpp: 实现文件
//

#include "pch.h"
#include "sshcm.h"
#include "afxdialogex.h"
#include "CAddHostDlg.h"
#include <afx.h>
#include <wincrypt.h>
#include <vector>
#include <strsafe.h>
#pragma comment(lib, "advapi32.lib")

CString CalcMD5String(const CString& input)
{
	// 转为 UTF-8
	int nBytes = WideCharToMultiByte(CP_UTF8, 0, input, -1, NULL, 0, NULL, NULL);
	if (nBytes <= 1)
		return _T("");

	std::vector<char> utf8(nBytes - 1);
	WideCharToMultiByte(CP_UTF8, 0, input, -1, utf8.data(), nBytes - 1, NULL, NULL);

	// 初始化 CryptoAPI
	HCRYPTPROV hProv = NULL;
	HCRYPTHASH hHash = NULL;
	BYTE rgbHash[16];
	DWORD cbHash = 16;
	TCHAR szHash[33] = { 0 };

	if (!CryptAcquireContext(&hProv, NULL, NULL, PROV_RSA_FULL, CRYPT_VERIFYCONTEXT))
		return _T("");

	if (!CryptCreateHash(hProv, CALG_MD5, 0, 0, &hHash))
	{
		CryptReleaseContext(hProv, 0);
		return _T("");
	}

	// 计算哈希
	if (!CryptHashData(hHash, reinterpret_cast<BYTE*>(utf8.data()), (DWORD)utf8.size(), 0))
	{
		CryptDestroyHash(hHash);
		CryptReleaseContext(hProv, 0);
		return _T("");
	}

	if (CryptGetHashParam(hHash, HP_HASHVAL, rgbHash, &cbHash, 0))
	{
		for (DWORD i = 0; i < cbHash; i++)
		{
			CString tmp;
			tmp.Format(_T("%02x"), rgbHash[i]);
			_tcscat_s(szHash, tmp);
		}
	}

	CryptDestroyHash(hHash);
	CryptReleaseContext(hProv, 0);

	return szHash;
}

// CAddHostDlg 对话框

IMPLEMENT_DYNAMIC(CAddHostDlg, CDialogEx)

CAddHostDlg::CAddHostDlg(CWnd* pParent /*=nullptr*/)
	: CDialogEx(IDD_SSH_ADD_DLG, pParent)
	, m_strName(_T(""))
	, m_strHost(_T(""))
	, m_strPort(_T(""))
	, m_strUser(_T(""))
	, m_strPass(_T(""))
	, m_strKey(_T(""))
	, m_bEditMode(FALSE)
	, m_strOldName(_T(""))
{

}

CAddHostDlg::~CAddHostDlg()
{
}

void CAddHostDlg::DoDataExchange(CDataExchange* pDX)
{
	CDialogEx::DoDataExchange(pDX);
	DDX_Text(pDX, IDC_EDIT_NAME, m_strName);

	DDX_Text(pDX, IDC_EDIT_HOST, m_strHost);
	DDX_Text(pDX, IDC_EDIT_PORT, m_strPort);
	DDX_Text(pDX, IDC_EDIT_USER, m_strUser);
	DDX_Text(pDX, IDC_EDIT_PASS, m_strPass);
	DDX_Text(pDX, IDC_EDIT_KEY, m_strKey);
}


BEGIN_MESSAGE_MAP(CAddHostDlg, CDialogEx)
	ON_BN_CLICKED(IDOK, &CAddHostDlg::OnBnClickedOk)
	ON_BN_CLICKED(IDC_BUTTON_BROWSE, &CAddHostDlg::OnBnClickedButtonBrowse)
	ON_BN_CLICKED(IDC_BUTTON2, &CAddHostDlg::OnBtnClearKeyPath)
END_MESSAGE_MAP()


// CAddHostDlg 消息处理程序

void CAddHostDlg::OnBnClickedOk()
{
	// TODO: 在此添加控件通知处理程序代码
	UpdateData(TRUE); // 同步输入框内容

	if (m_strName.IsEmpty() || m_strHost.IsEmpty())
	{
		AfxMessageBox(_T("名称和主机地址不能为空！"));
		return;
	}

	CDialogEx::OnOK();
}

void CAddHostDlg::OnBnClickedButtonBrowse()
{
	// TODO: 在此添加控件通知处理程序代码
	CFileDialog dlg(TRUE, _T("pem"), NULL,
		OFN_FILEMUSTEXIST | OFN_PATHMUSTEXIST,
		_T("私钥文件 (*.pem;*.key)|*.pem;*.key|所有文件 (*.*)|*.*||"), this);

	if (dlg.DoModal() == IDOK)
	{
		// 将选择的文件路径保存到变量
		UpdateData(TRUE);
		m_strKey = dlg.GetPathName();

		// 刷新到界面控件
		UpdateData(FALSE);
	}
}

BOOL CAddHostDlg::OnInitDialog()
{
	CDialogEx::OnInitDialog();

	// TODO:  在此添加额外的初始化
	
	// 设置窗口标题
	if (m_bEditMode)
		SetWindowText(_T("编辑主机配置"));
	else
		SetWindowText(_T("添加新主机"));

	UpdateData(FALSE);

	return TRUE;  // return TRUE unless you set the focus to a control
	// 异常: OCX 属性页应返回 FALSE
}

void CAddHostDlg::OnBtnClearKeyPath()
{
	// TODO: 在此添加控件通知处理程序代码
	CString input = _T("556901-828924");
	CString md5 = CalcMD5String(input);

	CString msg;
	msg.Format(_T("MD5(UTF8) = %s"), md5.GetString());
	AfxMessageBox(msg);


	UpdateData(TRUE);
	m_strKey = "";
	UpdateData(FALSE);
}
