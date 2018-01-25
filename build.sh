#!/bin/bash - 
#===============================================================================
#
#          FILE: build.sh
# 
#         USAGE: ./build.sh 
# 
#   DESCRIPTION: 
# 
#       OPTIONS: ---
#  REQUIREMENTS: ---
#          BUGS: ---
#         NOTES: ---
#        AUTHOR: YOUR NAME (), 
#  ORGANIZATION: 
#       CREATED: 21.11.2017 14:54
#      REVISION:  ---
#===============================================================================

set -o nounset                              # Treat unset variables as an error
TYPE=$[0]
if [ "$TYPE" == "ok" ]
then
    go build -o gdr main.go sources.go graph.go graphdata.go text.go
    if [ $? == 0 ]
    then
        mv gdr ~/bin/gdr
    else
        echo "build error!"
    fi
else
    go run main.go sources.go graph.go graphdata.go text.go
fi
