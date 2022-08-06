/*
 * @Author: jason chen (jasonchen@leansoftx.com, http://smallidea.cnblogs.com)
 * @Description:
 * @Date: 2021-11
 * @LastEditors: Jason Chen
 * @LastEditTime: 2022-08-02 21:50:36
 */
package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"path"

	//"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	scp "github.com/bramvdbogaerde/go-scp"
	"github.com/howeyc/gopass"
	"github.com/leansoftX/smartide-cli/internal/apk/i18n"
	"github.com/pkg/sftp"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
)

//
type SSHRemote struct {
	SSHHost        string
	SSHPort        int
	SSHUserName    string
	SSHPassword    string
	SSHKey         string
	SSHKeyPassword string
	SSHKeyPath     string
	Connection     *ssh.Client
}

var i18nInstance = i18n.GetInstance()

// 实例
func NewSSHRemote(host string, port int, userName, password string) (instance SSHRemote, err error) {

	instance = SSHRemote{}

	if (instance.Connection == &ssh.Client{}) || instance.Connection == nil {
		instance.SSHHost = host
		instance.SSHPort = port
		instance.SSHUserName = userName
		instance.SSHPassword = password

		connection, err := connectionDial(host, port, userName, password)
		if err != nil {
			return instance, err
		}

		instance.Connection = connection
	}

	return instance, nil
}

/*
 // 实例
 func (instance *SSHRemote) Instance(host string, port int, userName, password string) error {

	 if (instance.Connection == &ssh.Client{}) || instance.Connection == nil {
		 instance.SSHHost = host
		 instance.SSHPort = port
		 instance.SSHUserName = userName
		 instance.SSHPassword = password

		 connection, err := connectionDial(host, port, userName, password)
		 if err != nil {
			 return err
		 }

		 instance.Connection = connection
	 }

	 return nil
 } */

// 验证
func (instance *SSHRemote) CheckDail(host string, port int, userName, password string) error {

	if (instance.Connection == &ssh.Client{}) || instance.Connection == nil {

		connection, err := connectionDial(host, port, userName, password)

		if err != nil {
			return err
		}

		defer connection.Close()
	}

	return nil
}

// 判断端口是否可以（未被占用）
func (instance *SSHRemote) IsPortAvailable(port int) bool {
	command := fmt.Sprintf("sudo ss -tulwn | grep :%v", port)
	output, err := instance.ExeSSHCommand(command)
	if err != nil {
		if output != "" || err.Error() != "Process exited with status 1" {
			SmartIDELog.Error(err, output)
		}
	}

	return !strings.Contains(output, ":"+strconv.Itoa(port))
}

// 检查当前端口是否被占用，并返回一个可用端口
func (instance *SSHRemote) CheckAndGetAvailableRemotePort(checkPort int, step int) (usablePort int) {
	if step <= 0 {
		step = 100
	}
	usablePort = checkPort

	isPortUnable := false
	for !isPortUnable {

		if !instance.IsPortAvailable(usablePort) {
			usablePort += 100
		} else {
			isPortUnable = true
		}
	}

	return usablePort
}

// 获取远程主机上的当前目录
func (sshRemote *SSHRemote) GetRemotePwd() (currentDir string, err error) {
	currentDir, err = sshRemote.ExeSSHCommand("pwd")
	return currentDir, err
}

// 获取远程主机上的当前HOME目录
func (sshRemote *SSHRemote) GetRemoteHome() (currentDir string, err error) {
	currentDir, err = sshRemote.ExeSSHCommand("echo ${HOME}")
	return currentDir, err
}

//获取远程uid,gid
func (sshRemote *SSHRemote) GetRemoteUserInfo() (Uid string, Gid string) {
	remuid, err := sshRemote.ExeSSHCommand("id -u $USER")
	remgid, remgiderr := sshRemote.ExeSSHCommand("id -g $USER")
	SmartIDELog.Debug("Remote---Uid:" + remuid)
	SmartIDELog.Debug("Remote---Gid:" + remgid)

	if remuid != "" && err == nil {
		Uid = remuid
	} else {
		Uid = "1000"
	}
	if remgid != "" && remgiderr == nil {
		Gid = remgid
	} else {
		Gid = "1000"
	}
	return Uid, Gid
}

// 当前目录是否已经clone
func (instance *SSHRemote) IsCloned(workSpaceDir string) bool {
	gitDirPath := strings.Replace(FilePahtJoin4Linux(workSpaceDir, ".git"), "~/", "", -1) // 把路径变成 “a/b/c” 的形式，不支持 “./a/b/c”、“～/a/b/c”、“./a/b/c”
	cloneCommand := fmt.Sprintf(`[[ -d "%v" ]] && echo "1" || echo "0"`,
		gitDirPath)
	outContent, err := instance.ExeSSHCommand(cloneCommand)
	CheckError(err)

	// .git 文件夹不存在，清空文件夹
	if outContent == "0" {
		instance.ExeSSHCommand("sudo rm -rf " + workSpaceDir)
	}

	return outContent == "1"
}

// 文件是否存在
func (instance *SSHRemote) IsFileExist(filepath string) bool {

	filepath = instance.ConvertFilePath(filepath)

	command := fmt.Sprintf(`[[ -f "%v" ]] && echo "1" || echo "0"`, filepath)
	outContent, err := instance.ExeSSHCommand(command)
	CheckError(err)

	return outContent == "1"
}

// 文件是否存在
func (instance *SSHRemote) IsDirExist(filepath string) bool {

	filepath = instance.ConvertFilePath(filepath)

	command := fmt.Sprintf(`[[ -d "%v" ]] && echo "1" || echo "0"`, filepath)
	outContent, err := instance.ExeSSHCommand(command)
	CheckError(err)

	return outContent == "1"
}

// 文件是否存在
func (instance *SSHRemote) IsDirEmpty(dirPath string) bool {

	dirPath = instance.ConvertFilePath(dirPath)

	command := fmt.Sprintf(`[ "$(sudo ls -A %v)" ] && echo "0" || echo "111111"`, dirPath)
	//e.g. ls: cannot access '/home/localadmin/project/test001'111111\n: No such file or directory
	outContent, err := instance.ExeSSHCommand(command)
	CheckError(err)

	return strings.Contains(outContent, "111111") || strings.Contains(outContent, "No such file or directory")
}

// 清空文件夹
func (instance *SSHRemote) Clear(dirPath string) bool {
	dirPath = instance.ConvertFilePath(dirPath)

	command := fmt.Sprintf(`cd %v && sudo rm -rf {,.[!.],..?}*`, dirPath)
	_, err := instance.ExeSSHCommand(command)
	CheckError(err)

	return true
}

// 清空文件夹
func (instance *SSHRemote) Remove(fileOrDirPath string) bool {
	fileOrDirPath = instance.ConvertFilePath(fileOrDirPath)

	command := fmt.Sprintf(`sudo rm -rf %v`, fileOrDirPath)
	_, err := instance.ExeSSHCommand(command)
	CheckError(err)

	return true
}

// 复制本地文件夹中的文件到 远程主机对应的目录下
func (instance *SSHRemote) CopyDirectory(srcDirPath string, remoteDestDirPath string) error {
	remoteDestDirPath = instance.ConvertFilePath(remoteDestDirPath)

	//检测目录正确性
	if srcInfo, err := os.Stat(srcDirPath); err != nil {
		return err
	} else {
		if !srcInfo.IsDir() {
			return fmt.Errorf("在本地 %v 不是一个正确的目录！", srcDirPath)
		}
	}

	isExist := instance.IsDirExist(remoteDestDirPath)
	if !isExist {
		return fmt.Errorf("在远程主机上 %v 不是一个正确的目录！", remoteDestDirPath)
	}

	err := filepath.Walk(srcDirPath, func(localFilePath string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if !f.IsDir() {
			fielRelativePath := strings.Replace(localFilePath, srcDirPath, "", -1)
			remoteFilePath := FilePahtJoin4Linux(remoteDestDirPath, fielRelativePath)
			if instance.IsFileExist(remoteFilePath) {
				return fmt.Errorf("%v 文件已经存在！", remoteFilePath)
			}
			//	content, _ := ioutil.ReadFile(localFilePath)
			/* command := fmt.Sprintf(`echo '%v' >> %v`, content, remoteFilePath)
			_, err := instance.ExeSSHCommand(command) */

			/* err = instance.CheckAndCreateDir(filepath.Dir(remoteFilePath))
			 if err != nil {
				 instance.Clear(remoteDestDirPath)
				 return err
			 } */

			err = instance.CopyFile(localFilePath, remoteFilePath)
			if err != nil {
				instance.Clear(remoteDestDirPath)
				return err
			}
		}
		return nil
	})
	return err
}

// 获取文件内容
func (instance *SSHRemote) GetContent(filepath string) string {

	filepath = instance.ConvertFilePath(filepath)

	command := fmt.Sprintf(`cat "%v"`, filepath)
	outContent, err := instance.ExeSSHCommand(command)
	CheckError(err)

	return outContent
}

// 创建文件，如果存在就附加内容
func (sshRemote *SSHRemote) CreateFileByEcho(filepath string, content string) error {

	filepath = sshRemote.ConvertFilePath(filepath)

	/* 	// 检查并创建文件夹
	dir := path.Dir(filepath)
	err := sshRemote.CheckAndCreateDir(dir)
	if err != nil {
		return err
	} */

	// 创建文件
	content = strings.ReplaceAll(content, "\"", "\\\"")
	command := fmt.Sprintf(`sudo echo "%v" >> %v`, content, filepath)
	_, err := sshRemote.ExeSSHCommand(command)

	return err
}

// 检查并创建文件夹
func (sshRemote *SSHRemote) CheckAndCreateDir(dir string) error {
	dir = sshRemote.ConvertFilePath(dir)

	command := fmt.Sprintf("[[ -f \"%v\" ]] && echo \"1\" || mkdir -p \"%v\"", dir, dir)
	_, err := sshRemote.ExeSSHCommand(command)
	return err

}

// 转换文件路径为远程主机支持的
func (instance *SSHRemote) ConvertFilePath(filepath string) (newFilepath string) {
	newFilepath = filepath

	newFilepath = strings.ReplaceAll(filepath, "\\", "/")

	index := strings.Index(newFilepath, "~/")
	if index == 0 {
		pwd, err := instance.GetRemotePwd()
		CheckError(err)
		newFilepath = path.Join(pwd, strings.Replace(newFilepath, "~/", "", -1))
	}

	return newFilepath
}

// 检测远程服务器的环境，是否安装docker、docker-compose、git
func (instance *SSHRemote) CheckRemoteEnv() error {
	var errMsg []string

	//1. 环境监测
	//1.1. GIT
	output, err := instance.ExeSSHCommand("git version")
	if err != nil || strings.Contains(strings.ToLower(output), "error:") {
		if err != nil {
			SmartIDELog.Importance(i18nInstance.Main.Err_env_git_check, err.Error(), output)
		}
		return errors.New("请检查当前环境是否满足要求，参考：https://smartide.cn/zh/docs/install/docker/linux/")
	}

	//1.2. docker
	output, err = instance.ExeSSHCommand("docker version")
	if err != nil || strings.Contains(strings.ToLower(output), "error:") {
		if err != nil {
			SmartIDELog.Importance(i18nInstance.Main.Err_env_docker, err.Error(), output)
		}
		return errors.New("请检查当前环境是否满足要求，参考：https://smartide.cn/zh/docs/install/docker/linux/")
	}

	//1.3. docker-compose
	output, err = instance.ExeSSHCommand("docker-compose version")
	if err != nil ||
		(!strings.Contains(strings.ToLower(output), "docker-compose version") && !strings.Contains(strings.ToLower(output), "docker compose version")) ||
		strings.Contains(strings.ToLower(output), "error:") {
		if err != nil {
			SmartIDELog.Importance(i18nInstance.Main.Err_env_Docker_Compose, err.Error(), output)
		}
		return errors.New("请检查当前环境是否满足要求，参考：https://smartide.cn/zh/docs/install/docker/linux/")
	}

	//1.4. 默认的shell 是否为bash
	output, err = instance.ExeSSHCommand("echo $SHELL")
	if err != nil || !strings.Contains(output, "/bash") {
		if err != nil {
			SmartIDELog.Warning(err.Error())
		}
		SmartIDELog.Debug(output)
		return errors.New("请检查当前环境是否使用bash作为默认shell，参考：https://smartide.cn/zh/docs/install/docker/linux/")
	}

	//2. 错误判断
	if len(errMsg) > 0 {
		return errors.New(strings.Join(errMsg, "\\n "))
	}

	// 把当前用户加到docker用户组里面
	_, err = instance.ExeSSHCommand("sudo usermod -a -G docker " + instance.SSHUserName)
	if err != nil {
		SmartIDELog.Debug(err.Error())
	}

	// clone 代码库时，不提示：“are you sure you want to continue connecting (yes/no) ”
	sshConfig, err := instance.ExeSSHCommand("[[ -f \".ssh/config\" ]] && cat ~/.ssh/config || echo \"\"")
	if err != nil {
		return err
	}
	if !strings.Contains(sshConfig, "StrictHostKeyChecking no") { // 不包含就添加
		command := "if [ ! -d ～/.ssh ]; then mkdir -p ~/.ssh; fi && echo -e \"StrictHostKeyChecking no\n\" >> ~/.ssh/config"
		_, err := instance.ExeSSHCommand(command)
		if err != nil {
			return err
		}
	}

	return nil
}

// git clone
func (instance *SSHRemote) GitClone(gitRepoUrl string, workSpaceDir string, no string, cmd *cobra.Command) error {

	fflags := cmd.Flags()
	userName, _ := fflags.GetString(Flags_ServerUserName)
	if instance.IsCloned(workSpaceDir) {
		SmartIDELog.Info(i18n.GetInstance().Common.Info_gitrepo_cloned)
		return nil
	}

	if strings.TrimSpace(gitRepoUrl) == "" {
		SmartIDELog.Error(i18n.GetInstance().Common.Err_sshremote_param_repourl_none)
	}
	if workSpaceDir == "" {
		workSpaceDir = GetRepoName(gitRepoUrl)
	}

	// 检测是否为ssh模式
	// 是否覆盖服务器上的私钥文件
	// 文件存在时提示是否覆盖
	// 获取工作区策略中的 ssh-key
	// workspace.GetWSPolicies(no, "2")
	// 读取本地的ssh配置文件
	// 读取本地的 id_rsa 文件
	// , string(localRsaPub)
	// 公钥 文件不同时才会提示覆盖
	// fmt.Scanln(&isOverwrite)
	// 提示私钥文件是否覆盖（不覆盖就无法执行git clone）
	// fmt.Scanln(&isAllowCopyPrivateKey)
	// 读取本地的 id_rsa 文件
	// 读取本地的 id_rsa.pub 文件
	// 执行私钥文件复制
	// log
	// 执行私钥密码的取消 —— 把私钥密码设置为空
	// https://docs.github.com/cn/authentication/connecting-to-github-with-ssh/working-with-ssh-key-passphrases
	// instance.ExecSSHkeyPolicy(gitRepoUrl, no, cmd)

	// 执行clone
	//gitDirPath := strings.Replace(FilePahtJoin4Linux(workSpaceDir, ".git"), "~/", "", -1) // 把路径变成 “a/b/c” 的形式，不支持 “./a/b/c”、“～/a/b/c”、“./a/b/c”
	GIT_SSH_COMMAND := fmt.Sprintf(`GIT_SSH_COMMAND='ssh -i ~/.ssh/id_rsa_%s_%s -o IdentitiesOnly=yes'`, userName, no)

	cloneCommand := fmt.Sprintf(`%s git clone %v %v`,
		GIT_SSH_COMMAND, gitRepoUrl, workSpaceDir) // .git 文件如果不存在，在需要git clone
	err := instance.ExecSSHCommandRealTimeFunc(cloneCommand, func(output string) error {
		if strings.Contains(output, "error") || strings.Contains(output, "fatal") {

			newGitRepoUrl := strings.ToLower(gitRepoUrl)

			// 需要录入密码的情况
			if strings.Contains(output, "could not read Password for") { // 常规录入密码
				SmartIDELog.Console(i18n.GetInstance().Common.Info_please_enter_password)
				passwordBytes, _ := gopass.GetPasswdMasked()
				password := string(passwordBytes)

				// 添加密码到 https/http 链接中
				index := strings.LastIndex(newGitRepoUrl, "@")
				if index < 0 {
					newGitRepoUrl = strings.Replace(newGitRepoUrl, "https://", "https://"+password+"@", -1)
					newGitRepoUrl = strings.Replace(newGitRepoUrl, "http://", "http://"+password+"@", -1)
				} else {
					header := newGitRepoUrl[:strings.Index(newGitRepoUrl, "//")+2]
					newGitRepoUrl = header + password + newGitRepoUrl[index:]
				}
				SmartIDELog.Debug(newGitRepoUrl)

				// 再次运行 git clone
				instance.ExecSSHCommandRealTimeFunc(cloneCommand, nil)

			} else {
				return errors.New(output)
			}

		} /* else {
			SmartIDELog.ConsoleInLine(output)
			if strings.Contains(output, "done.") {
				fmt.Println()
			}
		} */

		return nil
	})

	// log
	if err == nil {
		SmartIDELog.Info(i18n.GetInstance().Common.Info_gitrepo_clone_done)
	}

	return err
}

func (instance *SSHRemote) ExecSSHkeyPolicy(no string, cmd *cobra.Command) {

	isOverwrite := "y"
	isAllowCopyPrivateKey := ""
	fflags := cmd.Flags()
	userName, _ := fflags.GetString(Flags_ServerUserName)
	commandRsa := fmt.Sprintf(`[[ -f ".ssh/id_rsa_%s_%s" ]] && cat ~/.ssh/id_rsa_%s_%s || echo ""`, userName, no, userName, no)
	remoteRsaPri, err := instance.ExeSSHCommandConsole(commandRsa, false)
	CheckError(err)
	SmartIDELog.DebugF("%v >> `%v`", commandRsa, "****")

	commandRsaPub := fmt.Sprintf(`[[ -f ".ssh/id_rsa.pub_%s_%s" ]] && cat ~/.ssh/id_rsa.pub_%s_%s || echo ""`, userName, no, userName, no)
	remoteRsaPub, err := instance.ExeSSHCommandConsole(commandRsaPub, false)
	CheckError(err)
	SmartIDELog.DebugF("%v >> `%v`", commandRsaPub, "****")

	idRsa := ""
	idRsaPub := ""
	var ws []WorkspacePolicy
	if no != "" {
		ws, err = GetWSPolicies(no, "2", cmd)
		CheckError(err)
	}

	if len(ws) > 0 {
		idRsa = ws[len(ws)-1].IdRsA
		idRsaPub = ws[len(ws)-1].IdRsAPub
	}
	// 远程公私钥都不为空
	if strings.ReplaceAll(remoteRsaPri, "\n", "") != "" && strings.ReplaceAll(remoteRsaPub, "\n", "") != "" {
		localRsaPub := ""
		if no != "" {
			isAllowCopyPrivateKey = "y"
			isOverwrite = "y"
			localRsaPub = idRsaPub

		} else {
			homeDir, err := os.UserHomeDir()
			CheckError(err)
			rsaPub, err := ioutil.ReadFile(filepath.Join(homeDir, "/.ssh/id_rsa.pub"))
			localRsaPub = string(rsaPub)
			CheckError(err)

		}
		//公钥文件不同时才会提示覆盖 非sever 模式
		if strings.TrimSpace(remoteRsaPub) != strings.TrimSpace(string(localRsaPub)) && no == "" {
			SmartIDELog.Console(i18n.GetInstance().Common.Info_privatekey_is_overwrite)
			fmt.Scanln(&isOverwrite)
			// isOverwrite = "y"

			//公钥文件相同时 非sever 不覆盖
		} else if strings.TrimSpace(remoteRsaPub) == strings.TrimSpace(string(localRsaPub)) && no == "" {
			isOverwrite = "n"
		} else { //公钥文件不同、公钥文件相同server模式 都是直接覆盖
			SmartIDELog.Debug(i18n.GetInstance().Common.Debug_same_not_overwrite)
			isOverwrite = "y"
		}

		/* 		if no == "" { // 远程公私钥有至少有一个为空的 非server模式
			SmartIDELog.Console(i18n.GetInstance().Common.Info_whether_overwrite)
			//isAllowCopyPrivateKey = "y"
			fmt.Scanln(&isAllowCopyPrivateKey)
		} */

	}

	if isAllowCopyPrivateKey == "y" || isOverwrite == "y" {

		if isOverwrite == "y" {

			if no == "" {
				if homeDir, err := os.UserHomeDir(); err == nil {
					if rsa, err := ioutil.ReadFile(filepath.Join(homeDir, "/.ssh/id_rsa")); err == nil {
						idRsa = string(rsa)
					}
					if rsaPub, err := ioutil.ReadFile(filepath.Join(homeDir, "/.ssh/id_rsa.pub")); err == nil {
						idRsaPub = string(rsaPub)
					}
				}

			}
			if idRsa != "" && idRsaPub != "" {
				command := fmt.Sprintf(`mkdir -p .ssh
									chmod 700 -R ~/.ssh
									rm -rf ~/.ssh/id_rsa_%s_%s
									echo "%v" >> ~/.ssh/id_rsa_%s_%s
									chmod 600 ~/.ssh/id_rsa_%s_%s

									rm -rf ~/.ssh/id_rsa.pub_%s_%s
									echo "%v" >> ~/.ssh/id_rsa.pub_%s_%s
									chmod 600 ~/.ssh/id_rsa.pub_%s_%s

`, userName, no, string(idRsa), userName, no, userName, no, userName, no, string(idRsaPub), userName, no, userName, no)
				output, err := instance.ExeSSHCommandConsole(command, false)
				CheckError(err, output)

				consoleCommand := strings.ReplaceAll(command, string(idRsa), "***")
				consoleCommand = strings.ReplaceAll(consoleCommand, string(idRsaPub), "***")
				SmartIDELog.DebugF("%v >> `%v`", consoleCommand, output)
				if no == "" {
					instance.sshSaveEmptyPassphrase()

				}
			}

		}
	}
}

// 保存一个空密码，保证后续的git clone不需要输入私钥的密码
func (instance *SSHRemote) sshSaveEmptyPassphrase() {
	// 如果本身就是空密码，就不需要执行了
	output, _ := instance.ExeSSHCommand("ssh-keygen -f ~/.ssh/id_rsa -p")
	if !strings.Contains(output, "Enter old passphrase") {
		return
	}

	session, err := instance.Connection.NewSession()
	CheckError(err)
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	err = session.RequestPty("xterm", 80, 40, modes)
	CheckError(err)

	stdoutB := new(bytes.Buffer)
	session.Stdout = stdoutB
	in, _ := session.StdinPipe()

	go func(in io.Writer, output *bytes.Buffer) {

		var t int = 0

		for {
			str := string(output.Bytes()[t:])
			if str == "" {
				continue
			}

			t = output.Len()

			if strings.Contains(str, "Enter old passphrase") {
				SmartIDELog.Console(i18n.GetInstance().Common.Info_please_enter_password)

				password, err := gopass.GetPasswdMasked()
				CheckError(err)

				_, err = in.Write([]byte(string(password) + "\n"))
				CheckError(err)
			} else if strings.Contains(str, "Enter new passphrase (empty for no passphrase)") {
				_, err = in.Write([]byte("\n"))
				CheckError(err)
			} else if strings.Contains(str, "Enter same passphrase again") {
				_, err = in.Write([]byte("\n"))
				CheckError(err)
				SmartIDELog.Info(i18nInstance.Common.Info_ssh_rsa_cancel_pwd_successed)
				break
			} else {
				SmartIDELog.Debug(str)
			}
		}
	}(in, stdoutB)

	err = session.Run("ssh-keygen -f ~/.ssh/id_rsa -p")
	CheckError(err)
}

// 从git clone地址中获取repo的名称
func GetRepoName(repoUrl string) string {
	index := strings.LastIndex(repoUrl, "/")
	return strings.Replace(repoUrl[index+1:], ".git", "", -1)
}

// 执行ssh command，在session模式下，standard output 只能在执行结束的时候获取到
func (instance *SSHRemote) ExeSSHCommand(sshCommand string) (outContent string, err error) {

	return instance.ExeSSHCommandConsole(sshCommand, true)
}

// 复制文件
func (instance *SSHRemote) CopyFile(localFilePath string, remoteFilepath string) error {
	var (
		err        error
		sftpClient *sftp.Client
	)

	// create sftp client
	if sftpClient, err = sftp.NewClient(instance.Connection); err != nil {
		return err
	}
	defer sftpClient.Close()

	//Local file path and folder on remote machine for testing
	srcFile, err := os.Open(localFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer srcFile.Close()

	// 创建目录
	parent := filepath.Dir(remoteFilepath)
	path := string(filepath.Separator)
	dirs := strings.Split(parent, path)
	for _, dir := range dirs {
		path = filepath.Join(path, dir)
		_ = sftpClient.Mkdir(path)
	}

	//
	dstFile, err := sftpClient.Create(remoteFilepath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	//
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	SmartIDELog.Debug(fmt.Sprintf("copy file (%v) to remote server (%v) finished!", localFilePath, remoteFilepath))
	return nil
}

// 执行ssh command，在session模式下，standard output 只能在执行结束的时候获取到
func (instance *SSHRemote) ExeSSHCommandConsole(sshCommand string, isConsoleAndLog bool) (outContent string, err error) {
	if len(sshCommand) <= 0 {
		return "", nil
	}

	session, err := instance.Connection.NewSession()
	CheckError(err)

	// 在ssh主机上执行命令
	SmartIDELog.Debug(fmt.Sprintf("SSH Console %v:%v -> %v ......", instance.SSHHost, instance.SSHPort, sshCommand))
	out, err := session.CombinedOutput(sshCommand)
	outContent = string(out)
	defer session.Close()

	// 空错误判断
	if err != nil {
		if outContent == "" && err.Error() == "Process exited with status 1" {
			SmartIDELog.Debug(i18n.GetInstance().Common.Debug_empty_error)
		}
	}

	// 记录日志，有些情况下不想输出信息，比如cat id_rsa时
	if isConsoleAndLog {
		outContent = strings.Trim(outContent, "\n")
		SmartIDELog.Debug(fmt.Sprintf("SSH Console %v:%v -> %v >> `%v`", instance.SSHHost, instance.SSHPort, sshCommand, outContent))
	}

	return outContent, err
}

// 实时执行
func (instance *SSHRemote) ExecSSHCommandRealTime(sshCommand string) (err error) {

	return instance.ExecSSHCommandRealTimeFunc(sshCommand, nil)
}

// 实时执行，带函数
func (instance *SSHRemote) ExecSSHCommandRealTimeFunc(sshCommand string, customExecuteFun func(output string) error) (err error) {

	SmartIDELog.Debug(fmt.Sprintf("SSH RealTime %v:%v -> %v", instance.SSHHost, instance.SSHPort, sshCommand))
	if (*instance == SSHRemote{}) {
		return errors.New(i18nInstance.Common.Err_ssh_dial_none)
	}

	session, err := instance.Connection.NewSession()
	CheckError(err)
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	err = session.RequestPty("xterm", 80, 40, modes)
	CheckError(err)

	//sshIn, _ := session.StdinPipe()
	sshOut, _ := session.StdoutPipe()

	originExecuteFun := func(output string) error {
		if strings.Contains(output, "error") || strings.Contains(output, "fatal") {
			return fmt.Errorf(output)
		} else {
			fmt.Print(output) // 进度信息不需要记录到日志
			if strings.Contains(output, "done.") {
				fmt.Println()
			}
		}
		return nil
	}

	chExit := make(chan bool)
	func1 := func() error { //in io.Writer, out io.Reader, exit chan bool
		for {
			isExit := false
			select {
			case <-chExit:
				isExit = true
			default:
			}

			if isExit { // 退出
				break
			}

			// https://gist.github.com/hivefans/ffeaf3964924c943dd7ed83b406bbdea#file-shell_output-go-L22
			buf := make([]byte, 1000)
			n, err := sshOut.Read(buf)
			if err != nil {
				SmartIDELog.Debug(err.Error())
			}
			originMsg := string(buf[:n])

			if originMsg == "" {
				continue
			}

			if originMsg == "" {
				continue
			}

			err = originExecuteFun(originMsg)
			if err != nil {
				return err
			}

			if customExecuteFun != nil {
				err = customExecuteFun(originMsg)
				if err != nil {
					return err
				}
			}

			/* array := strings.Split(originMsg, "\r\n")
			for _, sub := range array {
				if len(sub) == 0 || sub == "\r\n" { //|| sub == "\r"
					continue
				}

				err = originExecuteFun(sub)
				if err != nil {
					return err
				}

				if customExecuteFun != nil {
					err = customExecuteFun(sub)
					if err != nil {
						return err
					}
				}
			} */

		}
		return nil
	}

	group := new(errgroup.Group)
	group.Go(func1)

	err = session.Run(sshCommand)
	close(chExit)

	err2 := group.Wait()
	if (err != nil && os.IsNotExist(err)) && err2 != nil {
		SmartIDELog.ImportanceWithError(err2)
	} else {
		err = err2
	}
	fmt.Println()

	return err
}

/* func isChanClosed(ch chan bool) bool {
	if len(ch) == 0 {
		select {
		case _, ok := <-ch:
			return !ok
		}
	}
	return false
} */

func (instance *SSHRemote) RemoteUpload(filesMaps map[string]string) (err error) {
	// initialize SSH connection
	var clientConfig *ssh.ClientConfig

	if len(instance.SSHPassword) > 0 {

		if len(strings.TrimSpace(instance.SSHPassword)) == 0 {
			SmartIDELog.Error(i18nInstance.Common.Err_ssh_password_required)
		}

		clientConfig = &ssh.ClientConfig{
			User: instance.SSHUserName,
			Auth: []ssh.AuthMethod{
				ssh.Password(instance.SSHPassword),
			},
			Timeout: 30 * time.Second, // 30 秒超时
			// 解决 “ssh: must specify HostKeyCallback” 的问题
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return nil
			},
		}

	} else { // 如果用户不输入用户名和密码，则尝试使用ssh key pair的方式链接远程服务器
		//var hostKey ssh.PublicKey
		homePath, err := os.UserHomeDir()
		if err != nil {
			CheckError(err)
		}
		filePath := filepath.Join(homePath, "/.ssh/id_rsa")
		key, err := ioutil.ReadFile(filePath)
		CheckError(err, "unable to read private key:")

		// Create the Signer for this private key.
		signer, err := ssh.ParsePrivateKey(key)
		CheckError(err, "unable to parse private key:")

		clientConfig = &ssh.ClientConfig{
			User: instance.SSHUserName,
			Auth: []ssh.AuthMethod{
				// Use the PublicKeys method for remote authentication.
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				// use OpenSSH's known_hosts file if you care about host validation
				return nil
			},
		}

	}

	addr := fmt.Sprintf("%v:%v", instance.SSHHost, instance.SSHPort)

	if err == nil {
		for k, v := range filesMaps {

			client := scp.NewClient(addr, clientConfig)
			err = client.Connect()
			if err != nil {
				fmt.Println("Couldn't establish a connection to the remote server ", err)
				return
			}
			// Open a file
			f, _ := os.Open(k)

			defer client.Close()
			// Finaly, copy the file over
			// Usage: CopyFile(fileReader, remotePath, permission)
			defer f.Close()

			err = client.CopyFile(f, v, "0777")
			if err != nil {
				fmt.Println("Error while copying file ", err)
			}

		}

		// Close client connection after the file has been copied

	}
	return
}

// 连接到远程主机
func connectionDial(sshHost string, sshPort int, sshUserName, sshPassword string) (clientConn *ssh.Client, err error) {
	// initialize SSH connection
	var clientConfig *ssh.ClientConfig
	if sshPort <= 0 {
		sshPort = 22
	}

	if len(sshPassword) > 0 {

		if len(strings.TrimSpace(sshPassword)) == 0 {
			SmartIDELog.Error(i18n.GetInstance().Common.Err_password_none)
		}

		clientConfig = &ssh.ClientConfig{
			User: sshUserName,
			Auth: []ssh.AuthMethod{
				ssh.Password(sshPassword),
			},
			Timeout: 10 * time.Second, // 10 秒超时
			// 解决 “ssh: must specify HostKeyCallback” 的问题
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return nil
			},
		}

	} else { // 如果用户不输入用户名和密码，则尝试使用ssh key pair的方式链接远程服务器
		//var hostKey ssh.PublicKey
		homePath, err := os.UserHomeDir()
		CheckError(err)
		filePath := filepath.Join(homePath, "/.ssh/id_rsa")
		key, err := ioutil.ReadFile(filePath)
		CheckError(err, "unable to read private key:")

		// Create the Signer for this private key.
		signer, err := ssh.ParsePrivateKey(key)
		CheckError(err, "unable to parse private key:")

		clientConfig = &ssh.ClientConfig{
			User: sshUserName,
			Auth: []ssh.AuthMethod{
				// Use the PublicKeys method for remote authentication.
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				// use OpenSSH's known_hosts file if you care about host validation
				return nil
			},
		}

	}

	addr := fmt.Sprintf("%v:%v", sshHost, sshPort)
	return ssh.Dial("tcp", addr, clientConfig)
}

type GVA_MODEL struct {
	ID        uint      // 主键ID
	CreatedAt time.Time // 创建时间
	UpdatedAt time.Time // 更新时间
}
type WorkspacePolicy struct {
	GVA_MODEL
	Wid              string `json:"wid" form:"wid" `
	Name             string `json:"name" form:"name"`
	Status           *bool  `json:"status" form:"status" ` // 状态
	JustOne          *bool  `json:"justone" form:"justone" `
	Schdule          int    `json:"schdule" form:"schdule" `
	Type             int    `json:"type" form:"type" `
	Tasks            string `json:"tasks" form:"tasks" `
	OwnerGUID        string `json:"ownerGuid" form:"ownerGuid"`
	GitConifgContent string `json:"gitConfigContent" form:"gitConfigContent" `
	IdRsA            string `json:"id_rsa" form:"id_rsa"`
	IdRsAPub         string `json:"id_rsa_pub" form:"id_rsa_pub" `
}

type WSPolicyResponse struct {
	Code int `json:"code"`
	Data struct {
		Workspacepolicies []WorkspacePolicy `json:"list"`
	} `json:"data"`
	Msg string `json:"msg"`
}

const (
	Flags_ServerHost      = "serverhost"
	Flags_ServerToken     = "servertoken"
	Flags_ServerOwnerGuid = "serverownerguid"
	Flags_ServerUserName  = "serverusername"
)

func GetWSPolicies(no string, t string, cmd *cobra.Command) (ws []WorkspacePolicy, err error) {
	fflags := cmd.Flags()
	host, _ := fflags.GetString(Flags_ServerHost)
	token, _ := fflags.GetString(Flags_ServerToken)
	ownerGuid, _ := fflags.GetString(Flags_ServerOwnerGuid)
	var response = ""
	url := fmt.Sprint(host, "/api/smartide/workspacepolicy/getList")
	if response, err = Get(url, map[string]string{"ownerGuid": ownerGuid, "type": t}, map[string]string{"Content-Type": "application/json", "x-token": token}); response != "" {
		l := &WSPolicyResponse{}
		err = json.Unmarshal([]byte(response), l)
		if err = json.Unmarshal([]byte(response), l); err == nil {
			if l.Code == 0 && (len(l.Data.Workspacepolicies) != 0) {
				return l.Data.Workspacepolicies, err
			}
		}
	}
	return []WorkspacePolicy{}, err
}

// Add publickey to .ssh/authorized_keys file on remote host(vm mode)
func (instance *SSHRemote) AddPublicKeyIntoAuthorizedkeys() {
	execCommand := "[[ -f ~/.ssh/id_rsa.pub__ ]] && cat ~/.ssh/id_rsa.pub__ > ~/.ssh/authorized_keys__"
	output, err := instance.ExeSSHCommand(execCommand)
	if err != nil {
		if output != "" || err.Error() != "Process exited with status 1" {
			SmartIDELog.Error(err, output)
		}
	}
}
