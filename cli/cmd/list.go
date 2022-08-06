/*
 * @Author: jason chen (jasonchen@leansoftx.com, http://smallidea.cnblogs.com)
 * @Description:
 * @Date: 2021-11
 * @LastEditors: Jason Chen
 * @LastEditTime: 2022-06-07 15:37:39
 */
package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/leansoftX/smartide-cli/internal/biz/workspace"
	"github.com/leansoftX/smartide-cli/internal/dal"
	"github.com/leansoftX/smartide-cli/internal/model"
	"github.com/leansoftX/smartide-cli/pkg/common"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var listCmd = &cobra.Command{
	Use:     "list",
	Short:   i18nInstance.List.Info_help_short,
	Long:    i18nInstance.List.Info_help_long,
	Aliases: []string{"ls"},
	Example: `  smartide list`,
	Run: func(cmd *cobra.Command, args []string) {

		common.SmartIDELog.Info(i18nInstance.List.Info_start)
		cliRunningEnv := workspace.CliRunningEnvEnum_Client
		if value, _ := cmd.Flags().GetString("mode"); strings.ToLower(value) == "server" {
			cliRunningEnv = workspace.CliRunningEvnEnum_Server
		}
		printWorkspaces(cliRunningEnv)
		common.SmartIDELog.Info(i18nInstance.List.Info_end)
	},
}

// 打印 service 列表
func printWorkspaces(cliRunningEnv workspace.CliRunningEvnEnum) {
	workspaces, err := dal.GetWorkspaceList()
	common.CheckError(err)

	auth, err := workspace.GetCurrentUser()
	common.CheckError(err)
	if auth != (model.Auth{}) && auth.Token != "" {
		// 从api 获取workspace
		serverWorkSpaces, err := workspace.GetServerWorkspaceList(auth, cliRunningEnv)

		if err != nil { // 有错误仅给警告
			common.SmartIDELog.Importance("从服务器获取工作区列表失败，" + err.Error())
		} else { //
			workspaces = append(workspaces, serverWorkSpaces...)
		}
	}
	if len(workspaces) <= 0 {
		common.SmartIDELog.Info(i18nInstance.List.Info_dal_none)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	fmt.Fprintln(w, i18nInstance.List.Info_workspace_list_header)

	// 等于标题字符的长度
	tmpArray := strings.Split(i18nInstance.List.Info_workspace_list_header, "\t")
	outputArray := []string{}
	for _, str := range tmpArray {
		chars := ""
		for i := 0; i < len(str); i++ {
			chars += "-"
		}
		outputArray = append(outputArray, chars)
	}
	fmt.Fprintln(w, strings.Join(outputArray, "\t"))

	// 内容
	for _, worksapce := range workspaces {
		dir := worksapce.WorkingDirectoryPath
		if len(dir) <= 0 {
			dir = "-"
		}
		config := worksapce.ConfigFileRelativePath
		if len(config) <= 0 {
			config = "-"
		}
		local, _ := time.LoadLocation("Local")                                      // 北京时区
		createTime := worksapce.CreatedTime.In(local).Format("2006-01-02 15:04:05") // 格式化输出
		host := "-"
		if (worksapce.Remote != workspace.RemoteInfo{}) {
			host = fmt.Sprint(worksapce.Remote.Addr, ":", worksapce.Remote.SSHPort)
		}
		workspaceName := worksapce.Name
		if worksapce.ServerWorkSpace != nil {
			label := worksapce.ServerWorkSpace.Status.GetDesc()
			workspaceName = fmt.Sprintf("%v (%v)", workspaceName, label)
		}
		line := fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v\t%v", worksapce.ID, workspaceName, worksapce.Mode, dir, config, host, createTime)
		fmt.Fprintln(w, line)
	}
	w.Flush()
}

func init() {

}
