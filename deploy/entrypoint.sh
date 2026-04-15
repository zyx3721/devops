#!/bin/sh
#********************************************************************
# 项目名称：DevOps
# 文件名称：entrypoint.sh
# 创建时间：2026-04-14 22:33:28
#
# 系统用户：jerion
# 作　　者：Jerion
# 联系邮箱：416685476@qq.com
# 功能描述：容器启动脚本，初始化日志目录并启动 supervisord
#********************************************************************

set -e

# 创建持久化数据目录（挂载卷时目录可能不存在）
mkdir -p /app/data/logs

# 启动 supervisord
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
