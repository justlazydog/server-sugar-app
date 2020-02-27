#!/bin/bash
# shellcheck disable=SC2009
ps -ef|grep server-sugar-app|grep -v 'grep'|awk '{print $2}'|xargs kill -9
nohup ./server-sugar-app &
