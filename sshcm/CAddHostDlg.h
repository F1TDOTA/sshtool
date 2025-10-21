#pragma once
#include "afxdialogex.h"


// CAddHostDlg 对话框

class CAddHostDlg : public CDialogEx
{
	DECLARE_DYNAMIC(CAddHostDlg)

public:
	CAddHostDlg(CWnd* pParent = nullptr);   // 标准构造函数
	virtual ~CAddHostDlg();

// 对话框数据
#ifdef AFX_DESIGN_TIME
	enum { IDD = IDD_SSH_ADD_DLG };
#endif

protected:
	virtual void DoDataExchange(CDataExchange* pDX);    // DDX/DDV 支持

	DECLARE_MESSAGE_MAP()
public:
	afx_msg void OnBnClickedOk();
	CString m_strName;
	CString m_strHost;
	CString m_strPort;
	CString m_strUser;
	CString m_strPass;
	CString m_strKey;
	afx_msg void OnBnClickedButtonBrowse();
	BOOL m_bEditMode;
	CString m_strOldName;
	virtual BOOL OnInitDialog();
	afx_msg void OnBtnClearKeyPath();
};
