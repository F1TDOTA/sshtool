
// sshcmDlg.cpp: 实现文件
//

#include "pch.h"
#include "framework.h"
#include "sshcm.h"
#include "sshcmDlg.h"
#include "afxdialogex.h"
#include "CAddHostDlg.h"
#include <shlobj.h>
#include <tlhelp32.h>
#include <afxstr.h> 
#include <string>
#include <vector>
#include <iostream>
#include <fstream>
#include <sstream>

#ifdef _DEBUG
#define new DEBUG_NEW
#endif

CString GetAppDirectory(bool bExeDir = false)
{
	if (bExeDir)
	{
		TCHAR sz[MAX_PATH];
		GetModuleFileName(NULL, sz, _countof(sz));
		CString s = sz;
		int pos = s.ReverseFind(_T('\\'));
		if (pos >= 0) return s.Left(pos);
		return s;
	}
	else
	{
		TCHAR buf[MAX_PATH];
		if (GetCurrentDirectory(_countof(buf), buf) > 0)
		{
			CString cwd = buf;
			if (cwd.Right(1) == _T("\\")) cwd.TrimRight(_T("\\"));
			return cwd;
		}
		return _T("");
	}
}

void CsshcmDlg::LoadSshConfig(const CString& iniPath)
{
	CFileFind finder;
	if (!finder.FindFile(iniPath))
	{
		CString msg;
		msg.Format(_T("配置文件不存在！\n路径：%s"), iniPath);
		AfxMessageBox(msg, MB_ICONERROR);
		exit(0);
	}

	m_listHosts.DeleteAllItems();
	m_allHosts.clear();

	// 先读取[ssh] 节下的 hosts 字段
	TCHAR big_buf[4096];
	GetPrivateProfileString(_T("ssh"), _T("hosts"), _T(""), big_buf, 4096, iniPath);
	m_strHosts = big_buf;
	m_strHosts.Trim();

	// 如果是空的就直接返回
	if (m_strHosts.IsEmpty())
		return;

	// 按逗号分割主机名
	CString token;
	int curPos = 0;
	CStringList hostList;

	while (!(token = m_strHosts.Tokenize(_T(","), curPos)).IsEmpty())
	{
		token.Trim();
		if (!token.IsEmpty())
			hostList.AddTail(token);
	}

	// 遍历每个主机节，读取详细参数
	POSITION pos = hostList.GetHeadPosition();
	while (pos)
	{
		CString name = hostList.GetNext(pos);

		SshHost h;
		h.strName = name;

		TCHAR val[256];

		GetPrivateProfileString(name, _T("host"), _T(""), val, 256, iniPath);
		h.strHost = val;

		h.iPort = GetPrivateProfileInt(name, _T("port"), 22, iniPath);

		GetPrivateProfileString(name, _T("user"), _T(""), val, 256, iniPath);
		h.strUser = val;

		GetPrivateProfileString(name, _T("pass"), _T(""), val, 256, iniPath);
		h.strPass = val;

		GetPrivateProfileString(name, _T("private_key"), _T(""), val, 256, iniPath);
		h.strKeyPath = val;

		// 保存结果
		m_allHosts.push_back(h);  
	}

	// 读取外部程序配置
	TCHAR buf[512];

	GetPrivateProfileString(_T("external_prog"), _T("xshell_path"), _T(""), buf, 512, iniPath);
	m_externProg.xshellPath = buf;

	GetPrivateProfileString(_T("external_prog"), _T("xshell_conf_dir"), _T(""), buf, 512, iniPath);
	m_externProg.xshellConfDir = buf;

	GetPrivateProfileString(_T("external_prog"), _T("plink_path"), _T(""), buf, 512, iniPath);
	m_externProg.plinkPath = buf;

	GetPrivateProfileString(_T("external_prog"), _T("putty_path"), _T(""), buf, 512, iniPath);
	m_externProg.puttyPath = buf;

	GetPrivateProfileString(_T("external_prog"), _T("securecrt_path"), _T(""), buf, 512, iniPath);
	m_externProg.secureCrtPath = buf;

	GetPrivateProfileString(_T("external_prog"), _T("securecrt_conf_dir"), _T(""), buf, 512, iniPath);
	m_externProg.secureCrtConfDir = buf;

	GetPrivateProfileString(_T("external_prog"), _T("winscp_path"), _T(""), buf, 512, iniPath);
	m_externProg.winscpPath = buf;

	// 读取监控配置
	GetPrivateProfileString(_T("monitor"), _T("monitor_dir"), _T(""), buf, 512, iniPath);
	m_monitorConf.monitorDir = buf;

	GetPrivateProfileString(_T("monitor"), _T("upload_host"), _T(""), buf, 512, iniPath);
	m_monitorConf.uploadHost = buf;

	GetPrivateProfileString(_T("monitor"), _T("upload_path"), _T(""), buf, 512, iniPath);
	m_monitorConf.uploadPath = buf;
	

}

// 用于应用程序“关于”菜单项的 CAboutDlg 对话框

class CAboutDlg : public CDialogEx
{
public:
	CAboutDlg();

// 对话框数据
#ifdef AFX_DESIGN_TIME
	enum { IDD = IDD_ABOUTBOX };
#endif

	protected:
	virtual void DoDataExchange(CDataExchange* pDX);    // DDX/DDV 支持

// 实现
protected:
	DECLARE_MESSAGE_MAP()
};

CAboutDlg::CAboutDlg() : CDialogEx(IDD_ABOUTBOX)
{
}

void CAboutDlg::DoDataExchange(CDataExchange* pDX)
{
	CDialogEx::DoDataExchange(pDX);
}

BEGIN_MESSAGE_MAP(CAboutDlg, CDialogEx)
END_MESSAGE_MAP()


// CsshcmDlg 对话框



CsshcmDlg::CsshcmDlg(CWnd* pParent /*=nullptr*/)
	: CDialogEx(IDD_SSHCM_DIALOG, pParent)
	, m_strMonitorDir(_T(""))
	, m_strSearch(_T(""))
	, m_strProgName(_T("BuildRun.exe"))
	, m_strUploadPath(_T(""))
{
	m_hIcon = AfxGetApp()->LoadIcon(IDR_MAINFRAME);
}

void CsshcmDlg::DoDataExchange(CDataExchange* pDX)
{
	CDialogEx::DoDataExchange(pDX);
	DDX_Control(pDX, IDC_SSH_LIST, m_listHosts);
	DDX_Text(pDX, IDC_EDIT1, m_strMonitorDir);
	DDX_Text(pDX, IDC_EDIT_SEARCH, m_strSearch);
	DDX_Control(pDX, IDC_COMBO_SERVERS, m_comboServers);
	DDX_Control(pDX, IDC_BTN_START_SERVICE, m_btnStartStop);
	DDX_Text(pDX, IDC_EDIT_UPLOAD_PATH, m_strUploadPath);
	DDX_Control(pDX, IDC_EDIT_LOG, m_editLog);
	DDX_Control(pDX, IDC_STATIC_TOTAL, m_sshTotalText);
}

BEGIN_MESSAGE_MAP(CsshcmDlg, CDialogEx)
	ON_WM_SYSCOMMAND()
	ON_WM_PAINT()
	ON_WM_QUERYDRAGICON()
	ON_NOTIFY(LVN_ITEMCHANGED, IDC_SSH_LIST, &CsshcmDlg::OnLvnItemchangedSshList)
	ON_BN_CLICKED(IDC_BTN_ADD_SSH, &CsshcmDlg::OnBnClickedBtnAddSsh)
	ON_NOTIFY(NM_RCLICK, IDC_SSH_LIST, &CsshcmDlg::OnNMRClickSshList)

	ON_COMMAND(ID_32776, &CsshcmDlg::OnMenuDeleteConf)
	ON_COMMAND(ID_32774, &CsshcmDlg::OnMenuRefresh)
	ON_BN_CLICKED(IDC_BTN_REFRESH, &CsshcmDlg::OnBnClickedBtnRefresh)
	ON_BN_CLICKED(IDC_SELECT_MONITOR_DIR, &CsshcmDlg::OnBnClickedSelectMonitorDir)
	ON_EN_CHANGE(IDC_EDIT_SEARCH, &CsshcmDlg::OnEnChangeEditSearch)
	ON_BN_CLICKED(IDC_BTN_CLEAR, &CsshcmDlg::OnBnClickedBtnClear)
	ON_COMMAND(ID_32775, &CsshcmDlg::OnMenuEdit)
	ON_BN_CLICKED(IDC_BTN_START_SERVICE, &CsshcmDlg::OnBnClickedBtnStartService)
	ON_COMMAND(ID_OPEN_WINSCP, &CsshcmDlg::OnMenuOpenWinscp)
	ON_COMMAND(ID_OPEN_XSHELL, &CsshcmDlg::OnMenuOpenXshell)
	ON_COMMAND(ID_32785, &CsshcmDlg::OnMenuOpenPlink)
	ON_COMMAND(ID_32786, &CsshcmDlg::OnMenuOpenSecureCrt)
	ON_COMMAND(ID_32783, &CsshcmDlg::OnMenuOpenPutty)
	ON_BN_CLICKED(IDC_BTN_CLEAR_MONITOR_DIR, &CsshcmDlg::OnBnClickedBtnClearMonitorDir)
	ON_BN_CLICKED(IDC_BTN_SAVE_MONITOR, &CsshcmDlg::OnBnClickedBtnSaveMonitor)
	ON_BN_CLICKED(IDC_BTN_REFRESH_MONITOR, &CsshcmDlg::OnBnClickedBtnRefreshMonitor)
	ON_NOTIFY(NM_CLICK, IDC_SSH_LIST, &CsshcmDlg::OnNMClickSshList)
	ON_STN_CLICKED(IDC_STATIC_TOTAL, &CsshcmDlg::OnStnClickedStaticTotal)
END_MESSAGE_MAP()


// CsshcmDlg 消息处理程序

BOOL CsshcmDlg::OnInitDialog()
{
	CDialogEx::OnInitDialog();

	// 将“关于...”菜单项添加到系统菜单中。

	// IDM_ABOUTBOX 必须在系统命令范围内。
	ASSERT((IDM_ABOUTBOX & 0xFFF0) == IDM_ABOUTBOX);
	ASSERT(IDM_ABOUTBOX < 0xF000);

	CMenu* pSysMenu = GetSystemMenu(FALSE);
	if (pSysMenu != nullptr)
	{
		BOOL bNameValid;
		CString strAboutMenu;
		bNameValid = strAboutMenu.LoadString(IDS_ABOUTBOX);
		ASSERT(bNameValid);
		if (!strAboutMenu.IsEmpty())
		{
			pSysMenu->AppendMenu(MF_SEPARATOR);
			pSysMenu->AppendMenu(MF_STRING, IDM_ABOUTBOX, strAboutMenu);
		}
	}

	// 设置此对话框的图标。  当应用程序主窗口不是对话框时，框架将自动
	//  执行此操作
	SetIcon(m_hIcon, TRUE);			// 设置大图标
	SetIcon(m_hIcon, FALSE);		// 设置小图标

	// TODO: 在此添加额外的初始化代码
	m_listHosts.SetExtendedStyle(LVS_EX_FULLROWSELECT | LVS_EX_GRIDLINES);
	LoadIniToList();

	// 增加SSH服务器到下拉列表
	LoadServerToCombo();

	// 加载监控配置
	LoadMonitorConf();

	// 日志框初始化
	m_editLog.SubclassDlgItem(IDC_EDIT_LOG, this);
	m_editLog.SetLimitText(0);

	return TRUE;  // 除非将焦点设置到控件，否则返回 TRUE
}

void CsshcmDlg::OnSysCommand(UINT nID, LPARAM lParam)
{
	if ((nID & 0xFFF0) == IDM_ABOUTBOX)
	{
		CAboutDlg dlgAbout;
		dlgAbout.DoModal();
	}
	else
	{
		CDialogEx::OnSysCommand(nID, lParam);
	}
}

// 如果向对话框添加最小化按钮，则需要下面的代码
//  来绘制该图标。  对于使用文档/视图模型的 MFC 应用程序，
//  这将由框架自动完成。

void CsshcmDlg::OnPaint()
{
	if (IsIconic())
	{
		CPaintDC dc(this); // 用于绘制的设备上下文

		SendMessage(WM_ICONERASEBKGND, reinterpret_cast<WPARAM>(dc.GetSafeHdc()), 0);

		// 使图标在工作区矩形中居中
		int cxIcon = GetSystemMetrics(SM_CXICON);
		int cyIcon = GetSystemMetrics(SM_CYICON);
		CRect rect;
		GetClientRect(&rect);
		int x = (rect.Width() - cxIcon + 1) / 2;
		int y = (rect.Height() - cyIcon + 1) / 2;

		// 绘制图标
		dc.DrawIcon(x, y, m_hIcon);
	}
	else
	{
		CDialogEx::OnPaint();
	}
}

//当用户拖动最小化窗口时系统调用此函数取得光标
//显示。
HCURSOR CsshcmDlg::OnQueryDragIcon()
{
	return static_cast<HCURSOR>(m_hIcon);
}


void CsshcmDlg::OnLvnItemchangedSshList(NMHDR* pNMHDR, LRESULT* pResult)
{
	LPNMLISTVIEW pNMLV = reinterpret_cast<LPNMLISTVIEW>(pNMHDR);
	// TODO: 在此添加控件通知处理程序代码
	*pResult = 0;
}

void CsshcmDlg::AutoAdjustColumnWidth()
{
	if (!m_listHosts.GetSafeHwnd())
		return;

	CRect rcClient;
	m_listHosts.GetClientRect(&rcClient);

	CHeaderCtrl* pHeader = m_listHosts.GetHeaderCtrl();
	if (!pHeader)
		return;

	int nColumnCount = pHeader->GetItemCount();
	if (nColumnCount == 0)
		return;

	// 获取客户区宽度（减去滚动条）
	int nTotalWidth = rcClient.Width() - ::GetSystemMetrics(SM_CXVSCROLL);

	// ✅ 定义每列宽度百分比（总和 = 100）
	// 顺序对应：名称, 主机, 端口, 用户名, 密码, 私钥路径
	const double colPercents[] = { 0.15, 0.15, 0.10, 0.15, 0.20, 0.25 };
	const int nDefinedCols = sizeof(colPercents) / sizeof(colPercents[0]);

	if (nColumnCount != nDefinedCols)
	{
		TRACE(_T("警告: 列数(%d)与定义列数(%d)不匹配！\n"), nColumnCount, nDefinedCols);
		return;
	}

	m_listHosts.SetRedraw(FALSE);

	int usedWidth = 0;
	for (int i = 0; i < nColumnCount; ++i)
	{
		int nWidth = (int)(nTotalWidth * colPercents[i]);
		usedWidth += nWidth;

		// 最后一列补齐剩余空间
		if (i == nColumnCount - 1)
		{
			nWidth += (nTotalWidth - usedWidth);
		}

		m_listHosts.SetColumnWidth(i, nWidth);
	}

	m_listHosts.SetRedraw(TRUE);
}

void CsshcmDlg::DisplayHosts(const std::vector<SshHost>& hosts)
{
	m_listHosts.DeleteAllItems();

	for (int i = 0; i < (int)hosts.size(); ++i)
	{
		const auto& h = hosts[i];
		CString portStr;
		portStr.Format(_T("%d"), h.iPort);

		int nItem = m_listHosts.InsertItem(i, h.strName);
		m_listHosts.SetItemText(nItem, 1, h.strHost);
		m_listHosts.SetItemText(nItem, 2, portStr);
		m_listHosts.SetItemText(nItem, 3, h.strUser);
		m_listHosts.SetItemText(nItem, 4, h.strPass);
		m_listHosts.SetItemText(nItem, 5, h.strKeyPath);
	}
}

void CsshcmDlg::FillDataToGrid()
{
	CString keyword = m_strSearch.Trim();
	keyword.MakeLower();

	// 如果输入为空，恢复全部列表
	if (keyword.IsEmpty())
	{
		DisplayHosts(m_allHosts);
		return;
	}

	// 执行过滤
	std::vector<SshHost> filtered;
	for (const auto& h : m_allHosts)
	{
		CString text;
		text.Format(_T("%s %s %d %s %s %s"),
			h.strName, h.strHost, h.iPort, h.strUser, h.strPass, h.strKeyPath);

		CString lower = text;
		lower.MakeLower();

		if (lower.Find(keyword) >= 0)
		{
			filtered.push_back(h);
		}
	}

	DisplayHosts(filtered);
}

void CsshcmDlg::LoadServerToCombo()
{
	m_comboServers.ResetContent();
	m_comboServers.AddString(_T("请选择"));

	for (int i = 0; i < (int)m_allHosts.size(); ++i)
	{
		const SshHost& h = m_allHosts[i];
		CString item;
		item.Format(_T("[%s]-[%s:%d]"), h.strName, h.strHost, h.iPort);

		int idx = m_comboServers.AddString(item);
		m_comboServers.SetItemData(idx, (DWORD_PTR)(i+1)); // 保存索引
	}

	m_comboServers.SetCurSel(0);
}


void CsshcmDlg::LoadIniToList()
{
	m_iniPath = _T(".\\config.ini");
	// 重新加载数据
	LoadSshConfig(m_iniPath);

	// 删除所有列（从最后一个往前删）
	int nColCount = m_listHosts.GetHeaderCtrl()->GetItemCount();
	for (int i = nColCount - 1; i >= 0; --i)
	{
		m_listHosts.DeleteColumn(i);
	}

	// 设置列标题
	m_listHosts.InsertColumn(0, _T("名称"), LVCFMT_LEFT, 100);
	m_listHosts.InsertColumn(1, _T("主机"), LVCFMT_LEFT, 120);
	m_listHosts.InsertColumn(2, _T("端口"), LVCFMT_LEFT, 60);
	m_listHosts.InsertColumn(3, _T("用户名"), LVCFMT_LEFT, 80);
	m_listHosts.InsertColumn(4, _T("密码"), LVCFMT_LEFT, 80);
	m_listHosts.InsertColumn(5, _T("私钥路径"), LVCFMT_LEFT, 220);

	// 填充数据
	FillDataToGrid();
	AutoAdjustColumnWidth();

	CString msg;
	msg.Format(_T("总计：%d条"), (int)m_allHosts.size());
	m_sshTotalText.SetWindowText(msg);
}


BOOL CsshcmDlg::IsDuplicateName(const CString& newName, const CString& excludeName)
{
	for (auto& h : m_allHosts)
	{
		// 排除自己
		if (!excludeName.IsEmpty() && h.strName.CompareNoCase(excludeName) == 0)
			continue;

		if (h.strName.CompareNoCase(newName) == 0)
			return TRUE;
	}

	return FALSE;
}

BOOL CsshcmDlg::IsDuplicateHost(const CString& host, const CString& port,
		BOOL bEditMode, const CString& oldHost, const CString& oldPort)
{
	int nPort = _ttoi(port);
	int nOldPort = _ttoi(oldPort);
	size_t nLen = m_allHosts.size();

	for (size_t i = 0; i < nLen; ++i)
	{
		const SshHost& item = m_allHosts[i];

		// 编辑模式排除自己
		if (bEditMode && item.strHost.CompareNoCase(oldHost) == 0 && item.iPort == nOldPort)
			continue;

		if (item.strHost.CompareNoCase(host) == 0 && item.iPort == nPort)
			return TRUE;
	}

	return FALSE;
}

void CsshcmDlg::OnBnClickedBtnAddSsh()
{
	// TODO: 在此添加控件通知处理程序代码
	
	CAddHostDlg dlg;
	if (dlg.DoModal() == IDOK)
	{
		CString name = dlg.m_strName.Trim();
		CString host = dlg.m_strHost.Trim();
		CString port = dlg.m_strPort.Trim();
		CString user = dlg.m_strUser.Trim();
		CString pass = dlg.m_strPass.Trim();
		CString key = dlg.m_strKey.Trim();

		if (ValidateHostConfig(name, host, port, user) == FALSE)
		{
			return;
		}

		// 检查名称是否重复
		if (IsDuplicateName(name))
		{
			CString msg;
			msg.Format(_T("名称 [%s] 已存在，不能重复添加！"), name);
			AfxMessageBox(msg, MB_ICONWARNING);
			return;
		}

		// 检查是否重复（host + port）
		if (IsDuplicateHost(host, port, FALSE, NULL, NULL))
		{
			CString msg;
			msg.Format(_T("主机 [%s:%s] 已存在，不能重复添加！"), host, port);
			AfxMessageBox(msg, MB_ICONWARNING);
			return;
		}

		// 添加索引
		CString newHostsString;
		newHostsString.Format(_T("%s, %s"), m_strHosts, name);
		WritePrivateProfileString(_T("ssh"), _T("hosts"), newHostsString, m_iniPath);
		
		// 写入 INI 文件
		WritePrivateProfileString(name, _T("host"), host, m_iniPath);
		WritePrivateProfileString(name, _T("port"), port, m_iniPath);
		WritePrivateProfileString(name, _T("user"), user, m_iniPath);
		WritePrivateProfileString(name, _T("pass"), pass, m_iniPath);
		WritePrivateProfileString(name, _T("private_key"), key, m_iniPath);

		CString msg;
		msg.Format(_T("成功添加主机 [%s:%s]\n"), host, port);
		OutputDebugString(msg);

		// 刷新列表
		LoadIniToList();
		LoadServerToCombo();
	}
}

void CsshcmDlg::OnNMRClickSshList(NMHDR* pNMHDR, LRESULT* pResult)
{
	LPNMITEMACTIVATE pNMItemActivate = reinterpret_cast<LPNMITEMACTIVATE>(pNMHDR);
	// TODO: 在此添加控件通知处理程序代码

	CPoint pt;
	GetCursorPos(&pt); // 获取鼠标位置（屏幕坐标）

	// 选中被点击的行
	CPoint ptClient = pt;
	m_listHosts.ScreenToClient(&ptClient);

	LVHITTESTINFO hit = { 0 };
	hit.pt = ptClient;
	int nItem = m_listHosts.SubItemHitTest(&hit);
	if (nItem >= 0)
	{
		m_listHosts.SetItemState(nItem, LVIS_SELECTED | LVIS_FOCUSED, LVIS_SELECTED | LVIS_FOCUSED);
	}

	// 加载菜单资源
	CMenu menu;
	menu.LoadMenu(IDR_MENU_RCLICK);

	// 获取第一个子菜单（右键菜单是 Popup 类型）
	CMenu* pPopup = menu.GetSubMenu(0);
	ASSERT(pPopup != nullptr);

	// 弹出菜单
	pPopup->TrackPopupMenu(TPM_LEFTALIGN | TPM_RIGHTBUTTON, pt.x, pt.y, this);

	*pResult = 0;

}

void CsshcmDlg::OnMenuDeleteConf()
{
	// TODO: 在此添加命令处理程序代码
	POSITION pos = m_listHosts.GetFirstSelectedItemPosition();
	if (pos == NULL)
	{
		AfxMessageBox(_T("请先选择要删除的主机。"));
		return;
	}

	int nItem = m_listHosts.GetNextSelectedItem(pos);
	CString sectionName = m_listHosts.GetItemText(nItem, 0); // 第一列为名称

	CString msg;
	msg.Format(_T("确定要删除 [%s] 吗？"), sectionName);
	if (AfxMessageBox(msg, MB_YESNO | MB_ICONQUESTION) != IDYES)
		return;


	// 1. 从界面删除
	m_listHosts.DeleteItem(nItem);

	// 2. 从 INI 删除对应 Section
	if (!m_iniPath.IsEmpty())
	{
		CString token;
		int curPos = 0;
		std::vector<CString> names;

		while (!(token = m_strHosts.Tokenize(_T(","), curPos)).IsEmpty())
		{
			token.Trim();
			if (!token.IsEmpty() && token.CompareNoCase(sectionName) != 0)
			{
				names.push_back(token);
			}
		}

		CString newHosts;
		for (size_t i = 0; i < names.size(); ++i)
		{
			newHosts += names[i];
			if (i + 1 < names.size())
				newHosts += _T(", ");
		}

		BOOL bOK_1 = WritePrivateProfileString(_T("ssh"), _T("hosts"), newHosts, m_iniPath);
		BOOL bOK_2 = WritePrivateProfileString(sectionName, NULL, NULL, m_iniPath);
		if (bOK_1 && bOK_2)
		{
			CString info;
			info.Format(_T("已从配置文件中删除 [%s]\n"), sectionName);
			AfxMessageBox(info, MB_OK);
			OutputDebugString(info);
		}
		else
		{
			CString err;
			err.Format(_T("删除 [%s] 失败！请检查文件权限。\n"), sectionName);
			OutputDebugString(err);
			AfxMessageBox(err, MB_ICONERROR);
		}
	}

	LoadIniToList();
	LoadServerToCombo();
}

void CsshcmDlg::OnMenuRefresh()
{
	// TODO: 在此添加命令处理程序代码
	UpdateData(TRUE);
	OutputDebugString(_T("OnMenuRefresh 刷新配置列表。\n"));
	LoadIniToList();
}


void CsshcmDlg::OnBnClickedBtnRefresh()
{
	// TODO: 在此添加控件通知处理程序代码
	UpdateData(TRUE);
	OutputDebugString(_T("OnBnClickedBtnRefresh 刷新配置列表。\n"));
	LoadIniToList();
}

void CsshcmDlg::OnBnClickedSelectMonitorDir()
{
	// TODO: 在此添加控件通知处理程序代码
	BROWSEINFO bi = { 0 };
	bi.hwndOwner = m_hWnd;
	bi.lpszTitle = _T("请选择一个目录：");
	bi.ulFlags = BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE;

	LPITEMIDLIST pidl = SHBrowseForFolder(&bi);
	if (pidl != NULL)
	{
		TCHAR szPath[MAX_PATH] = { 0 };
		if (SHGetPathFromIDList(pidl, szPath))
		{
			m_strMonitorDir = szPath;
			UpdateData(FALSE);  // 更新到界面控件
		}
		CoTaskMemFree(pidl);
	}
}


void CsshcmDlg::OnEnChangeEditSearch()
{
	// TODO:  如果该控件是 RICHEDIT 控件，它将不
	// 发送此通知，除非重写 CDialogEx::OnInitDialog()
	// 函数并调用 CRichEditCtrl().SetEventMask()，
	// 同时将 ENM_CHANGE 标志“或”运算到掩码中。

	// TODO:  在此添加控件通知处理程序代码
	// 从 Edit 控件读取关键字
	UpdateData(TRUE);
	LoadIniToList();
}

void CsshcmDlg::OnBnClickedBtnClear()
{
	// TODO: 在此添加控件通知处理程序代码
	m_strSearch = "";
	UpdateData(FALSE);
	LoadIniToList();
}

BOOL CsshcmDlg::ValidateHostConfig(const CString& name, const CString& host, const CString& port, const CString& strUser)
{
	// 名称与主机不能为空
	if (name.IsEmpty() || host.IsEmpty())
	{
		AfxMessageBox(_T("名称和主机地址不能为空！"), MB_ICONERROR);
		return FALSE;
	}

	// 端口不能为空
	if (port.IsEmpty())
	{
		AfxMessageBox(_T("端口号不能为空！"), MB_ICONERROR);
		return FALSE;
	}

	// 检查端口是否为数字
	for (int i = 0; i < port.GetLength(); i++)
	{
		if (!_istdigit(port[i]))
		{
			AfxMessageBox(_T("端口号必须是数字！"), MB_ICONERROR);
			return FALSE;
		}
	}

	// 转为整数检测范围
	int nPort = _ttoi(port);
	if (nPort <= 0 || nPort > 65535)
	{
		AfxMessageBox(_T("端口号必须在 1–65535 之间！"), MB_ICONERROR);
		return FALSE;
	}

	// 用户名不能为空
	if (strUser.IsEmpty())
	{
		AfxMessageBox(_T("用户名不能为空！"), MB_ICONERROR);
		return FALSE;
	}

	return TRUE;
}

void CsshcmDlg::OnMenuEdit()
{
	// TODO: 在此添加命令处理程序代码
	POSITION pos = m_listHosts.GetFirstSelectedItemPosition();
	if (!pos)
	{
		AfxMessageBox(_T("请先选择需要编辑的项"), MB_ICONWARNING);
		return;
	}

	int nItem = m_listHosts.GetNextSelectedItem(pos);

	CString name = m_listHosts.GetItemText(nItem, 0);
	CString host = m_listHosts.GetItemText(nItem, 1);
	CString port = m_listHosts.GetItemText(nItem, 2);
	CString user = m_listHosts.GetItemText(nItem, 3);
	CString pass = m_listHosts.GetItemText(nItem, 4);
	CString key = m_listHosts.GetItemText(nItem, 5);


	while (true)
	{
		CAddHostDlg dlg;
		dlg.m_bEditMode = TRUE;
		dlg.m_strOldName = name;
		dlg.m_strName = name;
		dlg.m_strHost = host;
		dlg.m_strPort = port;
		dlg.m_strUser = user;
		dlg.m_strPass = pass;
		dlg.m_strKey = key;

		if (dlg.DoModal() != IDOK)
		{
			break;
		}

		CString newName = dlg.m_strName.Trim();
		CString newHost = dlg.m_strHost.Trim();
		CString newPort = dlg.m_strPort.Trim();
		CString newUser = dlg.m_strUser.Trim();
		CString newPass = dlg.m_strPass.Trim();
		CString newKey = dlg.m_strKey.Trim();

		if (ValidateHostConfig(newName, newHost, newPort, newUser) == FALSE)
		{
			continue;
		}

		// 检查名称是否重复
		if (IsDuplicateName(newName, name))
		{
			CString msg;
			msg.Format(_T("名称 [%s] 已存在，不能重复添加！"), newName);
			AfxMessageBox(msg, MB_ICONWARNING);
			continue;
		}

		// 检查是否重复（host + port）
		if (IsDuplicateHost(newHost, newPort, TRUE, host, port))
		{
			CString msg;
			msg.Format(_T("主机 [%s:%s] 已存在，不能重复添加！"), newHost, newPort);
			AfxMessageBox(msg, MB_ICONWARNING);
			continue;
		}

		// 删除旧section
		if (newName.CompareNoCase(name) != 0)
		{
			WritePrivateProfileString(name, NULL, NULL, m_iniPath);
		}

		// 保留原位置，替换旧名称为新名称
		CString token;
		int curPos = 0;
		CStringArray arr;

		while (!(token = m_strHosts.Tokenize(_T(","), curPos)).IsEmpty())
		{
			token.Trim();
			if (token.IsEmpty())
				continue;

			if (token.CompareNoCase(name) == 0)
				token = newName;   // 替换旧名称
			arr.Add(token);
		}

		// 重新拼接
		CString newHostsString;
		for (int i = 0; i < arr.GetSize(); ++i)
		{
			newHostsString += arr[i];
			if (i + 1 < arr.GetSize())
				newHostsString += _T(", ");
		}

		WritePrivateProfileString(_T("ssh"), _T("hosts"), newHostsString, m_iniPath);

		WritePrivateProfileString(newName, _T("host"), newHost, m_iniPath);
		WritePrivateProfileString(newName, _T("port"), newPort, m_iniPath);
		WritePrivateProfileString(newName, _T("user"), newUser, m_iniPath);
		WritePrivateProfileString(newName, _T("pass"), newPass, m_iniPath);
		WritePrivateProfileString(newName, _T("private_key"), newKey, m_iniPath);

		// 刷新列表
		LoadIniToList();

		// 自动高亮新项
		int nCount = m_listHosts.GetItemCount();
		for (int i = 0; i < nCount; ++i)
		{
			if (m_listHosts.GetItemText(i, 0).CompareNoCase(newName) == 0)
			{
				m_listHosts.SetItemState(i, LVIS_SELECTED | LVIS_FOCUSED, LVIS_SELECTED | LVIS_FOCUSED);
				m_listHosts.EnsureVisible(i, FALSE);
				break;
			}
		}

		break;
	}

	// 重新加载列表
	LoadServerToCombo();
}

void CsshcmDlg::OnMenuOpenXshell()
{
	// TODO: 在此添加命令处理程序代码
	CString strXshellPath = m_externProg.xshellPath;
	if (strXshellPath.IsEmpty())
	{
		AfxMessageBox(_T("xshell 路径未设置，请先设置"), MB_ICONWARNING);
		return;
	}

	CFileStatus status;
	if (!CFile::GetStatus(strXshellPath, status))
	{
		AfxMessageBox(_T("xshell 文件不存在，请检查配置文件中的路径"));
		return;
	}

	POSITION pos = m_listHosts.GetFirstSelectedItemPosition();
	if (!pos) 
	{
		AfxMessageBox(_T("未选择项，请先选择"), MB_ICONWARNING);
		return;
	}
	int nItem = m_listHosts.GetNextSelectedItem(pos);

	CString name = m_listHosts.GetItemText(nItem, 0);
	CString host = m_listHosts.GetItemText(nItem, 1);
	CString port = m_listHosts.GetItemText(nItem, 2);
	CString user = m_listHosts.GetItemText(nItem, 3);
	CString pass = m_listHosts.GetItemText(nItem, 4);
	CString key = m_listHosts.GetItemText(nItem, 5);

	CString strArgs;
	strArgs.Format(_T(" ssh://%s:%s@%s:%s"), user, pass, host, port);
	
	// 调用外部程序
	HINSTANCE hRet = ShellExecute(
		NULL,
		_T("open"),        
		strXshellPath,
		strArgs,
		NULL,
		SW_SHOWNORMAL
	);

	if ((INT_PTR)hRet <= 32)
	{
		CString msg;
		msg.Format(_T("启动xshell程序失败！错误码：%ld"), (INT_PTR)hRet);
		AfxMessageBox(msg, MB_ICONERROR);
	}
}

void CsshcmDlg::AppendLog(const CString& text)
{
	if (!::IsWindow(m_editLog.m_hWnd))
		return;

	// 1. 获取当前时间戳
	SYSTEMTIME st;
	GetLocalTime(&st);
	CString timeStr;
	timeStr.Format(_T("[%02d:%02d:%02d.%03d] "),
		st.wHour, st.wMinute, st.wSecond, st.wMilliseconds);

	// 2. 格式化换行
	CString formatted = text;
	formatted.Replace(_T("\r\n"), _T("\n"));
	formatted.Replace(_T("\n"), _T("\r\n"));
	if (formatted.Right(2) != _T("\r\n"))
		formatted += _T("\r\n");

	formatted = timeStr + formatted;

	// 3. 获取旧内容并将新日志插在最前面
	CString oldText;
	m_editLog.GetWindowText(oldText);
	CString newContent = formatted + oldText;

	// 4. 更新控件内容并滚动到顶部
	m_editLog.SetRedraw(FALSE);
	m_editLog.SetWindowText(newContent);
	m_editLog.SetRedraw(TRUE);
	m_editLog.LineScroll(0);
	m_editLog.Invalidate(FALSE);
	m_editLog.UpdateWindow();

}

void CsshcmDlg::KillProcessByName(const CString& exeName)
{
	AppendLog(_T("开始杀老进程...\r\n"));

	// 通知读取线程停止
	m_bStop = true;

	// 关闭管道，让线程自然退出
	if (m_hOutRd) { CloseHandle(m_hOutRd); m_hOutRd = NULL; }
	if (m_hErrRd) { CloseHandle(m_hErrRd); m_hErrRd = NULL; }

	// 不再等待线程，也不关心 m_pReadThread 是否还在
	m_pReadThread = nullptr;

	// 杀掉旧进程
	HANDLE hSnap = CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0);
	if (hSnap == INVALID_HANDLE_VALUE)
		return;

	PROCESSENTRY32 pe{ sizeof(PROCESSENTRY32) };
	if (Process32First(hSnap, &pe))
	{
		do
		{
			if (_tcsicmp(pe.szExeFile, exeName) == 0)
			{
				HANDLE hProc = OpenProcess(PROCESS_TERMINATE, FALSE, pe.th32ProcessID);
				if (hProc)
				{
					TerminateProcess(hProc, 0);
					CloseHandle(hProc);
					AppendLog(_T("杀掉进程成功: ") + exeName + _T("\r\n"));
				}
			}
		} while (Process32Next(hSnap, &pe));
	}

	CloseHandle(hSnap);
}

void CsshcmDlg::StartProcessAndCapture(const CString& exePath)
{
	SECURITY_ATTRIBUTES sa{ sizeof(sa), NULL, TRUE };
	HANDLE hOutRd = NULL, hOutWr = NULL;
	HANDLE hErrRd = NULL, hErrWr = NULL;

	CreatePipe(&hOutRd, &hOutWr, &sa, 0);
	SetHandleInformation(hOutRd, HANDLE_FLAG_INHERIT, 0);
	CreatePipe(&hErrRd, &hErrWr, &sa, 0);
	SetHandleInformation(hErrRd, HANDLE_FLAG_INHERIT, 0);

	STARTUPINFO si{};
	PROCESS_INFORMATION pi{};
	si.cb = sizeof(si);
	si.dwFlags |= STARTF_USESTDHANDLES;
	si.hStdOutput = hOutWr;
	si.hStdError = hErrWr;
	si.hStdInput = NULL;

	AppendLog(_T("启动服务...\r\n"));
	if (!CreateProcess(NULL, (LPTSTR)(LPCTSTR)exePath,
		NULL, NULL, TRUE, CREATE_NO_WINDOW,
		NULL, NULL, &si, &pi))
	{
		AppendLog(_T("启动服务失败.\r\n"));
		return;
	}

	CloseHandle(hOutWr);
	CloseHandle(hErrWr);

	m_hProcess = pi.hProcess;
	m_hOutRd = hOutRd;
	m_hErrRd = hErrRd;
	m_bStop = false;

	// 启动日志读取线程
	m_pReadThread = AfxBeginThread([](LPVOID pParam)->UINT {
		CsshcmDlg* dlg = (CsshcmDlg*)pParam;
		char buf[1024];
		DWORD dwRead;
		CString line;

		while (!dlg->m_bStop)
		{
			DWORD avail = 0;
			if (!PeekNamedPipe(dlg->m_hOutRd, NULL, 0, NULL, &avail, NULL))
				break;

			if (avail > 0)
			{
				if (ReadFile(dlg->m_hOutRd, buf, sizeof(buf) - 1, &dwRead, NULL) && dwRead > 0)
				{
					buf[dwRead] = '\0';

					// --- UTF-8 → Unicode 转换 ---
					int wlen = MultiByteToWideChar(CP_UTF8, 0, buf, -1, NULL, 0);
					CStringW wstr;
					LPWSTR pwstr = wstr.GetBuffer(wlen);
					MultiByteToWideChar(CP_UTF8, 0, buf, -1, pwstr, wlen);
					wstr.ReleaseBuffer();

					dlg->AppendLog(CString(wstr));
				}
			}

			if (!PeekNamedPipe(dlg->m_hErrRd, NULL, 0, NULL, &avail, NULL))
				break;

			if (avail > 0)
			{
				if (ReadFile(dlg->m_hErrRd, buf, sizeof(buf) - 1, &dwRead, NULL) && dwRead > 0)
				{
					buf[dwRead] = '\0';

					int wlen = MultiByteToWideChar(CP_UTF8, 0, buf, -1, NULL, 0);
					CStringW wstr;
					LPWSTR pwstr = wstr.GetBuffer(wlen);
					MultiByteToWideChar(CP_UTF8, 0, buf, -1, pwstr, wlen);
					wstr.ReleaseBuffer();

					dlg->AppendLog(_T("[ERR] ") + CString(wstr));
				}
			}

			DWORD code = STILL_ACTIVE;
			if (dlg->m_hProcess)
			{
				GetExitCodeProcess(dlg->m_hProcess, &code);
				if (code != STILL_ACTIVE)
					break;
			}
			Sleep(100);
		}

		dlg->AppendLog(_T("日志读取线程失败.\r\n"));
		return 0;
	}, this);
}

void CsshcmDlg::OnBnClickedBtnStartService()
{
	// TODO: 在此添加控件通知处理程序代码
	// 弹出
	int result = AfxMessageBox(_T("需要全量同步目录么?"), MB_YESNO | MB_ICONQUESTION);
	// 判断用户选择的按钮
	if (result == IDYES)
	{
		WritePrivateProfileString(_T("monitor"), _T("init_sync_all"), _T("1"), m_iniPath);
	}
	else
	{
		WritePrivateProfileString(_T("monitor"), _T("init_sync_all"), _T("0"), m_iniPath);
	}

	// 重启服务
	CString strExePath = GetAppDirectory(true);
	CString goExePath;
	goExePath.Format(_T("%s\\%s"), strExePath, m_strProgName);

	m_editLog.SetSel(0, -1);      // 选中全部
	m_editLog.ReplaceSel(_T("")); // 用空字符串替换
	m_editLog.LineScroll(0);      // 滚动回顶部
	AppendLog(_T("重新启动服务...\r\n"));

	KillProcessByName(m_strProgName);
	Sleep(200);
	StartProcessAndCapture(goExePath);
}

void CsshcmDlg::OnMenuOpenWinscp()
{
	// TODO: 在此添加命令处理程序代码
	CString strWinscpPath = m_externProg.winscpPath;
	if (strWinscpPath.IsEmpty())
	{
		AfxMessageBox(_T("winscp 路径未设置，请先设置"), MB_ICONWARNING);
		return;
	}

	CFileStatus status;
	if (!CFile::GetStatus(strWinscpPath, status))
	{
		AfxMessageBox(_T("winscp 文件不存在，请检查配置文件中的路径"));
		return;
	}

	POSITION pos = m_listHosts.GetFirstSelectedItemPosition();
	if (!pos)
	{
		AfxMessageBox(_T("未选择项，请先选择"), MB_ICONWARNING);
		return;
	}
	int nItem = m_listHosts.GetNextSelectedItem(pos);

	CString name = m_listHosts.GetItemText(nItem, 0);
	CString host = m_listHosts.GetItemText(nItem, 1);
	CString port = m_listHosts.GetItemText(nItem, 2);
	CString user = m_listHosts.GetItemText(nItem, 3);
	CString pass = m_listHosts.GetItemText(nItem, 4);
	CString key = m_listHosts.GetItemText(nItem, 5);

	CString strArgs;
	strArgs.Format(_T(" scp://%s:%s@%s:%s/"), user, pass, host, port);

	// 调用外部程序
	HINSTANCE hRet = ShellExecute(
		NULL,
		_T("open"),
		strWinscpPath,
		strArgs,
		NULL,
		SW_SHOWNORMAL
	);

	if ((INT_PTR)hRet <= 32)
	{
		CString msg;
		msg.Format(_T("启动winscp程序失败！错误码：%ld"), (INT_PTR)hRet);
		AfxMessageBox(msg, MB_ICONERROR);
	}
}

void CsshcmDlg::OnMenuOpenPlink()
{
	// TODO: 在此添加命令处理程序代码
	CString strPlinkPath = m_externProg.plinkPath;
	if (strPlinkPath.IsEmpty())
	{
		AfxMessageBox(_T("Plink 路径未设置，请先设置"), MB_ICONWARNING);
		return;
	}

	CFileStatus status;
	if (!CFile::GetStatus(strPlinkPath, status))
	{
		AfxMessageBox(_T("Plink 文件不存在，请检查配置文件中的路径"));
		return;
	}

	POSITION pos = m_listHosts.GetFirstSelectedItemPosition();
	if (!pos)
	{
		AfxMessageBox(_T("未选择项，请先选择"), MB_ICONWARNING);
		return;
	}
	int nItem = m_listHosts.GetNextSelectedItem(pos);

	CString name = m_listHosts.GetItemText(nItem, 0);
	CString host = m_listHosts.GetItemText(nItem, 1);
	CString port = m_listHosts.GetItemText(nItem, 2);
	CString user = m_listHosts.GetItemText(nItem, 3);
	CString pass = m_listHosts.GetItemText(nItem, 4);
	CString key = m_listHosts.GetItemText(nItem, 5);

	CString plinkCmd;
	if (!pass.IsEmpty())
		plinkCmd.Format(_T("\"%s\" -ssh -t -batch %s@%s -P %s -pw %s"), strPlinkPath, user, host, port, pass);
	else
		plinkCmd.Format(_T("\"%s\" -ssh -t %s@%s -P %s"), strPlinkPath, user, host, port);

	CString params;
	params.Format(_T("/k %s"), plinkCmd);

	SHELLEXECUTEINFO sei = { 0 };
	sei.cbSize = sizeof(sei);
	sei.fMask = SEE_MASK_FLAG_NO_UI; // 遇到错误不弹出默认错误对话，可根据需改为 SEE_MASK_NOCLOSEPROCESS 等
	sei.hwnd = NULL;
	sei.lpVerb = _T("open");
	sei.lpFile = _T("cmd.exe");
	sei.lpParameters = params;
	sei.nShow = SW_SHOWNORMAL;

	if (!ShellExecuteEx(&sei))
	{
		DWORD err = GetLastError();
		CString msg;
		msg.Format(_T("启动Plink失败，错误码=%u"), err);
		AfxMessageBox(msg);
	}
	else
	{
		// 如果需要等待程序结束（阻塞当前线程）
		// WaitForSingleObject(sei.hProcess, INFINITE);

		// 不管是否等待，都要 CloseHandle 防止句柄泄露
		if (sei.hProcess)
		{
			CloseHandle(sei.hProcess);
			sei.hProcess = NULL;
		}
	}

}


void CsshcmDlg::OnMenuOpenSecureCrt()
{
	// TODO: 在此添加命令处理程序代码
	CString strSecureCrtPath = m_externProg.secureCrtPath;
	if (strSecureCrtPath.IsEmpty())
	{
		AfxMessageBox(_T("SecureCrt 路径未设置，请先设置"), MB_ICONWARNING);
		return;
	}

	CFileStatus status;
	if (!CFile::GetStatus(strSecureCrtPath, status))
	{
		AfxMessageBox(_T("SecureCrt 文件不存在，请检查配置文件中的路径"));
		return;
	}

	POSITION pos = m_listHosts.GetFirstSelectedItemPosition();
	if (!pos)
	{
		AfxMessageBox(_T("未选择项，请先选择"), MB_ICONWARNING);
		return;
	}
	int nItem = m_listHosts.GetNextSelectedItem(pos);

	CString name = m_listHosts.GetItemText(nItem, 0);
	CString host = m_listHosts.GetItemText(nItem, 1);
	CString port = m_listHosts.GetItemText(nItem, 2);
	CString user = m_listHosts.GetItemText(nItem, 3);
	CString pass = m_listHosts.GetItemText(nItem, 4);
	CString key = m_listHosts.GetItemText(nItem, 5);


	CString params;
	params.Format(L"/T /SSH2 /L %s /PASSWORD %s /P %s %s",
		(LPCTSTR)user,
		(LPCTSTR)pass,
		(LPCTSTR)port,
		(LPCTSTR)host);

	// ShellExecuteEx 需要 lpFile=exePath, lpParameters=params
	SHELLEXECUTEINFO sei = { 0 };
	sei.cbSize = sizeof(sei);
	sei.fMask = SEE_MASK_NOCLOSEPROCESS; // 若需要可以获取 hProcess
	sei.hwnd = NULL;
	sei.lpVerb = NULL; // "open"
	sei.lpFile = strSecureCrtPath; // 可给完整路径
	sei.lpParameters = params;
	sei.nShow = SW_SHOWNORMAL;

	if (!ShellExecuteEx(&sei))
	{
		DWORD err = GetLastError();
		CString msg;
		msg.Format(_T("启动SecureCrt失败，错误码=%u"), err);
		AfxMessageBox(msg);
	}
	else
	{
		// 如果需要等待程序结束（阻塞当前线程）
		// WaitForSingleObject(sei.hProcess, INFINITE);

		// 不管是否等待，都要 CloseHandle 防止句柄泄露
		if (sei.hProcess)
		{
			CloseHandle(sei.hProcess);
			sei.hProcess = NULL;
		}
	}
}

void CsshcmDlg::OnMenuOpenPutty()
{
	// TODO: 在此添加命令处理程序代码
	CString strPuttyPath = m_externProg.puttyPath;
	if (strPuttyPath.IsEmpty())
	{
		AfxMessageBox(_T("Putty 路径未设置，请先设置"), MB_ICONWARNING);
		return;
	}

	CFileStatus status;
	if (!CFile::GetStatus(strPuttyPath, status))
	{
		AfxMessageBox(_T("Putty 文件不存在，请检查配置文件中的路径"));
		return;
	}

	POSITION pos = m_listHosts.GetFirstSelectedItemPosition();
	if (!pos)
	{
		AfxMessageBox(_T("未选择项，请先选择"), MB_ICONWARNING);
		return;
	}
	int nItem = m_listHosts.GetNextSelectedItem(pos);

	CString name = m_listHosts.GetItemText(nItem, 0);
	CString host = m_listHosts.GetItemText(nItem, 1);
	CString port = m_listHosts.GetItemText(nItem, 2);
	CString user = m_listHosts.GetItemText(nItem, 3);
	CString pass = m_listHosts.GetItemText(nItem, 4);
	CString key = m_listHosts.GetItemText(nItem, 5);

	// 拼装，执行
	CString params;
	if (!pass.IsEmpty())
	{
		params.Format(L"-ssh -l %s -pw %s -P %s %s",
			(LPCTSTR)user,
			(LPCTSTR)pass,
			(LPCTSTR)port,
			(LPCTSTR)host);
	}
	else
	{
		params.Format(L"-ssh -i \"%s\" -l %s -P %s %s",
			(LPCTSTR)key,
			(LPCTSTR)user,
			(LPCTSTR)port,
			(LPCTSTR)host);
	}

	// 调试显示参数
	//AfxMessageBox(params);

	SHELLEXECUTEINFO sei = { 0 };
	sei.cbSize = sizeof(sei);
	sei.fMask = SEE_MASK_NOCLOSEPROCESS;
	sei.hwnd = NULL;
	sei.lpVerb = NULL;
	sei.lpFile = strPuttyPath;     // PuTTY 可执行文件路径
	sei.lpParameters = params;     // 拼好的参数
	sei.nShow = SW_SHOWNORMAL;

	if (!ShellExecuteEx(&sei))
	{
		DWORD err = GetLastError();
		CString msg;
		msg.Format(_T("启动PuTTY失败，错误码=%u"), err);
		AfxMessageBox(msg);
	}
	else
	{
		if (sei.hProcess)
		{
			CloseHandle(sei.hProcess);
			sei.hProcess = NULL;
		}
	}
}



void CsshcmDlg::OnBnClickedBtnClearMonitorDir()
{
	// TODO: 在此添加控件通知处理程序代码
	UpdateData(TRUE);
	m_strMonitorDir = "";
	UpdateData(FALSE);
}

void CsshcmDlg::OnBnClickedBtnSaveMonitor()
{
	// TODO: 在此添加控件通知处理程序代码
	UpdateData(TRUE);
	CString strSshServerName = _T("");

	int nSel = m_comboServers.GetCurSel();
	if (nSel != CB_ERR)
	{
		if (nSel != 0)
		{
			m_comboServers.GetLBText(nSel, strSshServerName);
		}
	}

	if (m_strMonitorDir.IsEmpty())
	{
		AfxMessageBox(_T("请先选择需要监控的文件目录"), MB_ICONWARNING);
		return;
	}

	if (m_strUploadPath.IsEmpty())
	{
		AfxMessageBox(_T("上传的文件路径未输入，请先输入"), MB_ICONWARNING);
		return;
	}

	CString strKey = _T("monitor");
	WritePrivateProfileString(strKey, _T("monitor_dir"), m_strMonitorDir, m_iniPath);
	WritePrivateProfileString(strKey, _T("upload_host"), strSshServerName, m_iniPath);
	WritePrivateProfileString(strKey, _T("upload_path"), m_strUploadPath, m_iniPath);

	AfxMessageBox(_T("配置保存成功，请重启服务"), MB_OK);
}


void CsshcmDlg::LoadMonitorConf()
{
	m_strMonitorDir = m_monitorConf.monitorDir;
	m_strUploadPath = m_monitorConf.uploadPath;

	CString target = m_monitorConf.uploadHost;
	int index = m_comboServers.FindStringExact(-1, target);
	if (index != CB_ERR)
	{
		m_comboServers.SetCurSel(index);
	}
	else
	{
		m_comboServers.SetCurSel(0);
	}

	UpdateData(FALSE);
}



void CsshcmDlg::OnBnClickedBtnRefreshMonitor()
{
	// TODO: 在此添加控件通知处理程序代码
	LoadIniToList();
	LoadMonitorConf();
}



void CsshcmDlg::CopyToClipboard(const CString& text)
{
	if (!OpenClipboard())
		return;

	EmptyClipboard();

	SIZE_T size = (text.GetLength() + 1) * sizeof(wchar_t);
	HGLOBAL hMem = GlobalAlloc(GMEM_MOVEABLE, size);
	if (hMem)
	{
		memcpy(GlobalLock(hMem), text.GetString(), size);
		GlobalUnlock(hMem);
		SetClipboardData(CF_UNICODETEXT, hMem);
	}

	CloseClipboard();
}

void CsshcmDlg::OnNMClickSshList(NMHDR* pNMHDR, LRESULT* pResult)
{
	LPNMITEMACTIVATE pNM = reinterpret_cast<LPNMITEMACTIVATE>(pNMHDR);

	// 是否点中有效行和列
	if (pNM->iItem < 0 || pNM->iSubItem < 0)
		return;

	const int PASSWORD_COL = 4; // 密码列索引
	if (pNM->iSubItem != PASSWORD_COL)
		return;

	// 从真实数据源获取密码
	CString password = m_listHosts.GetItemText(pNM->iItem, pNM->iSubItem);
	if (password.IsEmpty())
		return;

	CopyToClipboard(password);
	*pResult = 0;
	AfxMessageBox(_T("密码已复制到剪贴板"), MB_ICONINFORMATION);
}

void CsshcmDlg::OnStnClickedStaticTotal()
{
	// TODO: 在此添加控件通知处理程序代码
}
