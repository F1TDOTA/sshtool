
// sshcmDlg.h: 头文件
//
#pragma once

#include <vector>
#include <string>

typedef struct
{
	CString strName;
	CString strHost;
	int iPort;
	CString strUser;
	CString strPass;
	CString strKeyPath;
} SshHost;

typedef struct
{
	CString winscpPath;

	CString xshellPath;
	CString secureCrtPath;
	CString plinkPath;
	CString puttyPath;

	CString xshellConfDir;
	CString secureCrtConfDir;
} ExternalProgConfig;

typedef struct
{
	CString monitorDir;
	CString uploadHost;
	CString uploadPath;
} MonitorConfig;

// 自定义消息（确保不与其它消息冲突）
#define WM_GO_OUTPUT (WM_APP + 100)

// 传递给主窗口的数据结构（在堆上分配，主窗口收到后负责 delete）
struct GoOutputMsg
{
	std::string output;   // ANSI bytes，从子进程 stdout/stderr 读取到的原始字节
	DWORD exitCode;       // 进程退出码
};


// CsshcmDlg 对话框
class CsshcmDlg : public CDialogEx
{
// 构造
public:
	CsshcmDlg(CWnd* pParent = nullptr);	// 标准构造函数

// 对话框数据
#ifdef AFX_DESIGN_TIME
	enum { IDD = IDD_SSHCM_DIALOG };
#endif

	protected:
	virtual void DoDataExchange(CDataExchange* pDX);	// DDX/DDV 支持


// 实现
protected:
	HICON m_hIcon;

	// 生成的消息映射函数
	virtual BOOL OnInitDialog();
	afx_msg void OnSysCommand(UINT nID, LPARAM lParam);
	afx_msg void OnPaint();
	afx_msg HCURSOR OnQueryDragIcon();
	DECLARE_MESSAGE_MAP()

public:
	afx_msg void OnLvnItemchangedSshList(NMHDR* pNMHDR, LRESULT* pResult);
	afx_msg void OnBnClickedBtnAddSsh();
	void LoadIniToList();
	void LoadServerToCombo();
	void LoadMonitorConf();
	void LoadSshConfig(const CString& iniPath);
	BOOL IsDuplicateHost(const CString& host, const CString& port, BOOL bEditMode, const CString& oldHost, const CString& oldPort);
	BOOL IsDuplicateName(const CString& newName, const CString& excludeName = _T(""));
	BOOL ValidateHostConfig(const CString& name, const CString& host, const CString& port, const CString& strUser);
	void DisplayHosts(const std::vector<SshHost>& hosts);
	void FillDataToGrid();
	void AutoAdjustColumnWidth();
	CListCtrl m_listHosts;
	CString m_iniPath;
	afx_msg void OnNMRClickSshList(NMHDR* pNMHDR, LRESULT* pResult);
	afx_msg void OnMenuDeleteConf();
	afx_msg void OnMenuRefresh();
	afx_msg void OnBnClickedBtnRefresh();
	afx_msg void OnBnClickedSelectMonitorDir();
	CString m_strMonitorDir;
	CString m_strSearch;
	CString m_strHosts;
	std::vector<SshHost> m_allHosts;
	ExternalProgConfig m_externProg;
	MonitorConfig  m_monitorConf;
	afx_msg void OnEnChangeEditSearch();
	afx_msg void OnBnClickedBtnClear();
	afx_msg void OnMenuEdit();
	CComboBox m_comboServers;
	afx_msg void OnBnClickedBtnStartService();
	afx_msg void OnMenuOpenWinscp();
	afx_msg void OnMenuOpenXshell();
	CButton m_btnStartStop;
	CString m_strProgName;
	afx_msg void OnMenuOpenPlink();
	afx_msg void OnMenuOpenSecureCrt();
	afx_msg void OnMenuOpenPutty();
	afx_msg void OnBtnClearKeyPath();
	afx_msg void OnBnClickedBtnClearMonitorDir();
	afx_msg void OnBnClickedBtnSaveMonitor();
	CString m_strUploadPath;
	afx_msg void OnBnClickedBtnRefreshMonitor();
	CEdit m_editLog;
	void AppendLog(const CString& text);
	void KillProcessByName(const CString& exeName);
	void StartProcessAndCapture(const CString& exePath);
	HANDLE m_hProcess = nullptr;
	HANDLE m_hOutRd = nullptr;
	HANDLE m_hErrRd = nullptr;
	CWinThread* m_pReadThread = nullptr;
	volatile bool m_bStop = false;
};
