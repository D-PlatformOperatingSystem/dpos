#!/usr/bin/env bash

set -e
set -o pipefail
#set -o verbose
#set -o xtrace

# os: ubuntu16.04 x64
# first, you must install jq tool of json
# sudo apt-get install jq
# sudo apt-get install shellcheck, in order to static check shell script
# sudo apt-get install parallel

PWD=$(cd "$(dirname "$0")" && pwd)
export PATH="$PWD:$PATH"

CLI="./dplatformos-cli"

sedfix=""
if [ "$(uname)" == "Darwin" ]; then
    sedfix=".bak"
fi

dplatformosConfig="dplatformos.test.toml"
dplatformosBlockTime=1
autoTestCheckTimeout=10

function config_dplatformos() {

    # shellcheck disable=SC2015
    echo "# config dplatformos solo test"
    # update test environment
    sed -i $sedfix 's/^Title.*/Title="local"/g' ${dplatformosConfig}
    # grep -q '^TestNet' ${dplatformosConfig} && sed -i $sedfix 's/^TestNet.*/TestNet=true/' ${dplatformosConfig} || sed -i '/^Title/a TestNet=true' ${dplatformosConfig}

    if grep -q '^TestNet' ${dplatformosConfig}; then
        sed -i $sedfix 's/^TestNet.*/TestNet=true/' ${dplatformosConfig}
    else
        sed -i $sedfix '/^Title/a TestNet=true' ${dplatformosConfig}
    fi

    #update fee
    #    sed -i $sedfix 's/Fee=.*/Fee=100000/' ${dplatformosConfig}

    #update wallet store driver
    #    sed -i $sedfix '/^\[wallet\]/,/^\[wallet./ s/^driver.*/driver="leveldb"/' ${dplatformosConfig}
}

autotestConfig="autotest.toml"
autotestTempConfig="autotest.temp.toml"
function config_autotest() {

    echo "# config autotest"
    #delete all blank lines
    #    sed -i $sedfix '/^\s*$/d' ${autotestConfig}

    if [[ $1 == "" ]] || [[ $1 == "all" ]]; then
        cp ${autotestConfig} ${autotestTempConfig}
        sed -i $sedfix 's/^checkTimeout.*/checkTimeout='${autoTestCheckTimeout}'/' ${autotestTempConfig}
    else
        #copy config before [
        # sed -n '/^\[\[/!p;//q' ${autotestConfig} >${autotestTempConfig}
        #pre config auto test
        {

            echo 'cliCmd="./dplatformos-cli"'
            echo "checkTimeout=${autoTestCheckTimeout}"
        } >${autotestTempConfig}

        #specific dapp config
        for dapp in "$@"; do
            {
                echo "[[TestCaseFile]]"
                echo "dapp=\"$dapp\""
                echo "filename=\"$dapp.toml\""
            } >>${autotestTempConfig}

        done
    fi
}

function start_dplatformos() {

    echo "# start solo dplatformos"
    rm -rf ../local/datadir ../local/logs ../local/grpc.log
    ./dplatformos -f dplatformos.test.toml >/dev/null 2>&1 &
    local SLEEP=1
    sleep ${SLEEP}

    # query node run status
    ${CLI} block last_header

    echo "=========== # save seed to wallet ============="
    result=$(${CLI} seed save -p 1314domchain -s "tortoise main civil member grace happy century convince father cage beach hip maid merry rib" | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "save seed to wallet error seed, result: ${result}"
        exit 1
    fi

    echo "=========== # unlock wallet ============="
    result=$(${CLI} wallet unlock -p 1314domchain -t 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        exit 1
    fi

    echo "=========== # import private key returnAddr ============="
    result=$(${CLI} account import_key -k CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944 -l returnAddr | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi

    echo "=========== # import private key mining ============="
    result=$(${CLI} account import_key -k 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01 -l minerAddr | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi

    echo "=========== #transfer to miner addr ============="
    hash=$(${CLI} send coins transfer -a 10000 -n test -t 12oupcayRT7LvaC4qW4avxsTE7U41cKSio -k CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944)
    echo "${hash}"
    sleep ${dplatformosBlockTime}
    result=$(${CLI} tx query -s "${hash}" | jq '.receipt.tyName')
    if [[ ${result} != '"ExecOk"' ]]; then
        echo "Failed"
        ${CLI} tx query -s "${hash}" | jq '.' | cat
        exit 1
    fi

    echo "=========== #transfer to token amdin ============="
    hash=$(${CLI} send coins transfer -a 10 -n test -t 1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK -k CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944)

    echo "${hash}"
    sleep ${dplatformosBlockTime}
    result=$(${CLI} tx query -s "${hash}" | jq '.receipt.tyName')
    if [[ ${result} != '"ExecOk"' ]]; then
        echo "Failed"
        ${CLI} tx query -s "${hash}" | jq '.' | cat
        exit 1
    fi

    echo "=========== #config token blacklist ============="
    rawData=$(${CLI} config config_tx -c token-blacklist -o add -v BTC)
    signData=$(${CLI} wallet sign -d "${rawData}" -k 0xc34b5d9d44ac7b754806f761d3d4d2c4fe5214f6b074c19f069c4f5c2a29c8cc)
    hash=$(${CLI} wallet send -d "${signData}")

    echo "${hash}"
    sleep ${dplatformosBlockTime}
    result=$(${CLI} tx query -s "${hash}" | jq '.receipt.tyName')
    if [[ ${result} != '"ExecOk"' ]]; then
        echo "Failed"
        ${CLI} tx query -s "${hash}" | jq '.' | cat
        exit 1
    fi

}

function start_autotest() {

    echo "=========== #run autotest, make sure saving autotest.last.log file============="

    if [ -e autotest.log ]; then
        cat autotest.log >autotest.last.log
        rm autotest.log
    fi

    ./autotest -f ${autotestTempConfig}

}

function stop_dplatformos() {

    rv=$?
    echo "=========== #stop dplatformos ============="
    ${CLI} close || true
    #wait close
    sleep ${dplatformosBlockTime}
    echo "==========================================local-auto-test-shell-end========================================================="
    exit ${rv}
}

function main() {
    echo "==========================================local-auto-test-shell-begin========================================================"
    config_autotest "$@"
    config_dplatformos
    start_dplatformos
    start_autotest

}

trap "stop_dplatformos" INT TERM EXIT

main "$@"
