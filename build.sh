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
go build
if [ $? == 0 ]
then
    mv gdr-go ~/bin/gdr
else
    echo "build error!"
fi
