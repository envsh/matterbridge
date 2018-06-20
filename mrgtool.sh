#!/bin/sh

# Makefile
# matterbridge.toml
# matbrg.tsbin
# matterbridge.toml.96BE1
# matbrg.tsbin.96BE1
# issue.txt
# toxbrg.sqlite3
# VERSION
# buildinfo.go
# mrgtool.sh

oldbr=withtox-ups1.10
newbr=withtox-ups1.11
set -x

# directories
# bridge/tox/
git checkout withtox-ups1.10
git branch -D withtox-ups1.11
git subtree split --prefix=bridge/tox/ -b toxbrg
git checkout -b withtox-ups1.11 v1.11.0
git config user.email mrgbot@help.cc
git config user.name mbot1
git subtree add --prefix=bridge/tox/ . toxbrg
git branch -D toxbrg
# exit
#

git checkout  $oldbr Makefile
git checkout  $oldbr matterbridge.toml
git checkout  $oldbr matbrg.tsbin
git checkout  $oldbr matterbridge.toml.96BE1
git checkout  $oldbr matbrg.tsbin.96BE1
git checkout  $oldbr issue.txt
git checkout  $oldbr toxbrg.sqlite3
git checkout  $oldbr VERSION
git checkout  $oldbr buildinfo.go
# git checkout  $oldbr mrgtool.sh

git commit -a -m "mrg raw files"

# conflict
# bridge/bridge.go --
# bridge/config/config.go
# matterbridge.go
# gateway/gateway.go
# bridge/irc/irc.go # for !ping command


# git checkout  $oldbr gateway/gateway.go  # add line:  "tox": btox.New,
# git checkout  $oldbr bridge/config/config.go # add line:  Tox map[string]Protocol
# git checkout  $oldbr bridge/bridge.go # --
# git checkout  $oldbr matterbridge.go # -- # add line: printBuildInfo(true)

# for read
echo "git checkout $oldbr gateway/gateway.go"
echo "git checkout $oldbr bridge/config/config.go"
echo "git checkout $oldbr matterbridge.go"

