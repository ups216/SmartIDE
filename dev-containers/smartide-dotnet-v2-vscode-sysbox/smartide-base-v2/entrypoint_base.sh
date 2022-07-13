#!/bin/bash
###
 # @Author: kenan
 # @Date: 2022-05-24 14:37:27
 # @LastEditors: kenan
 # @LastEditTime: 2022-05-24 15:12:20
 # @FilePath: /smartide/dev-containers/smartide-dotnet-v2-vscode-sysbox/smartide-base-v2/entrypoint_base.sh
 # @Description: 
 # 
 # Copyright (c) 2022 by kenanlu@leansoftx.com, All Rights Reserved. 
### 

USER_UID=${LOCAL_USER_UID:-1000}
USER_GID=${LOCAL_USER_GID:-1000}
USER_PASS=${LOCAL_USER_PASSWORD:-"smartide123.@IDE"}
USERNAME=smartide
echo "Starting with USER_UID : $USER_UID"
echo "Starting with USER_GID : $USER_GID"
echo "Starting with USER_PASS : $USER_PASS"

# root运行容器，容器里面一样root运行
if [ $USER_UID == '0' ]; then

    echo "-----root------Starting"

    USERNAMEROOT=root

    chown -R $USERNAMEROOT:$USERNAMEROOT /home/project
    chown -R $USERNAMEROOT:$USERNAMEROOT /home/opvscode

    export HOME=/root

    echo "root:$USER_PASS" | chpasswd
    echo "-----------Starting sshd"
    # 后面不加$@容器会自动退出
    exec /usr/sbin/sshd -D -e "$@"

else

    #非root运行，通过传入环境变量创建自定义用户的uid,gid
    echo "-----smartide------Starting"

    export HOME=/home/$USERNAME

    chown -R $USERNAME:$USERNAME /home/project
    chown -R $USERNAME:$USERNAME /home/opvscode
    chown -R $USERNAME:$USERNAME /home/$USERNAME/.ssh


    echo "root:$USER_PASS" | chpasswd
    echo "smartide:$USER_PASS" | chpasswd
    echo "-----smartide------Starting sshd"
    exec /usr/sbin/sshd -D -e "$@"

fi
