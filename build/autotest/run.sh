#!/usr/bin/env bash

set -e
set -o pipefail
#set -o verbose
#set -o xtrace

# os: ubuntu16.04 x64

DPOS_PATH=../../

function copyAutoTestConfig() {

    declare -a DplatformOSAutoTestDirs=("${DPOS_PATH}/system")
    echo "#copy auto test config to path \"$1\""
    local AutoTestConfigFile="$1/autotest.toml"

    #pre config auto test
    {

        echo 'cliCmd="./dplatformos-cli"'
        echo "checkTimeout=60"
    } >"${AutoTestConfigFile}"

    #copy all the dapp test case config file
    for rootDir in "${DplatformOSAutoTestDirs[@]}"; do

        if [[ ! -d ${rootDir} ]]; then
            continue
        fi

        testDirArr=$(find "${rootDir}" -type d -name autotest)

        for autotest in ${testDirArr}; do

            dapp=$(basename "$(dirname "${autotest}")")
            dappConfig=${autotest}/${dapp}.toml

            #make sure dapp have auto test config
            if [ -e "${dappConfig}" ]; then

                cp "${dappConfig}" "$1"/

                #add dapp test case config
                {
                    echo "[[TestCaseFile]]"
                    echo "dapp=\"$dapp\""
                    echo "filename=\"$dapp.toml\""
                } >>"${AutoTestConfigFile}"

            fi

        done
    done
}

function copyDplatformOS() {

    echo "# copy dplatformos bin to path \"$1\", make sure build dplatformos"
    cp ../dplatformos ../dplatformos-cli ../dplatformos.toml "$1"
    cp "${DPOS_PATH}"/cmd/dplatformos/dplatformos.test.toml "$1"
}

function copyAll() {

    dir="$1"
    #check dir exist
    if [[ ! -d ${dir} ]]; then
        mkdir "${dir}"
    fi
    cp autotest "${dir}"
    copyAutoTestConfig "${dir}"
    copyDplatformOS "${dir}"
    echo "# all copy have done!"
}

function main() {

    dir="$1"
    copyAll "$dir" && cd "$dir" && ./autotest.sh "${@:2}" && cd ../

}

main "$@"
