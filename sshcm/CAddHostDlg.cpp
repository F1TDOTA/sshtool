// CAddHostDlg.cpp: 实现文件
//

#include "pch.h"
#include "sshcm.h"
#include "afxdialogex.h"
#include "CAddHostDlg.h"


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
