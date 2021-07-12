#!/usr/bin/env bash

coinsTxHashs1=("")
#coinsTxHashs2=("")
coinsgStr=""
coinsRepeatTx=1 #
coinsTotalAmount1="0"
coinsTxAmount="5" #
#defaultTxFee="0.001"
coinsAddr1=""
coinsTxSign=("")

function initCoinsAccount() {

    name="${CLI}"
    label="coinstest1"
    createAccount "${name}" $label
    coinsAddr1=$coinsgStr

    sleep 1

    name="${CLI}"
    fromAdd="12oupcayRT7LvaC4qW4avxsTE7U41cKSio"
    showCoinsBalance "${name}" $fromAdd
    coinsTotalAmount1=$coinsgStr

    shouAccountList "${name}"
}

function genFirstChainCoinstx() {
    echo "======   coins   ======"
    name=$CLI
    echo "    ：${name}"

    for ((i = 0; i < coinsRepeatTx; i++)); do
        From="12oupcayRT7LvaC4qW4avxsTE7U41cKSio"
        to=$coinsAddr1
        note="coinstx"
        amount=$coinsTxAmount

        genCoinsTx "${name}" $From $to $note $amount
        coinsTxSign[$i]="${coinsgStr}"

        block_wait_timeout "${CLI}" 1 20

        #
        sendCoinsTxHash "${name}" "${coinsgStr}"
        coinsTxHashs1[$i]="${coinsgStr}"

        sleep 1
        height=$(${name} block last_header | jq ".height")
        printf '    %d         %s \n' $i "${height}"
    done
}

function genSecondChainCoinstx() {
    echo "======      ======"
    name=$CLI
    echo "    ：${name}"

    #
    for ((i = 0; i < ${#coinsTxSign[*]}; i++)); do
        #
        sign="${coinsTxSign[$i]}"
        sendCoinsTxHash "${name}" "${sign}"

        sleep 1
        height=$(${name} block last_header | jq ".height")
        printf '    %d         %s \n' $i "${height}"
    done
}

function checkCoinsResult() {
    name=$CLI

    echo "====================     docker    ================="

    totalCoinsTxAmount="0"

    for ((i = 0; i < ${#coinsTxHashs1[*]}; i++)); do
        txHash=${coinsTxHashs1[$i]}
        echo $txHash
        txQuery "${name}" $txHash
        result=$?
        if [ $result -eq 0 ]; then
            #coinsTxFee1=$(echo "$coinsTxFee1 + $defaultTxFee" | bc)
            totalCoinsTxAmount=$(echo "$totalCoinsTxAmount + $coinsTxAmount" | bc)
        fi
        sleep 1
    done

    sleep 1

    fromAdd="12oupcayRT7LvaC4qW4avxsTE7U41cKSio"
    showCoinsBalance "${name}" $fromAdd
    value1=$coinsgStr

    sleep 1

    fromAdd=$coinsAddr1
    showCoinsBalance "${name}" $fromAdd
    value2=$coinsgStr

    actTotal=$(echo "$value1 + $value2 " | bc)
    echo "${name}     ：$actTotal"

    #if [ `echo "${actTotal} > ${coinsTotalAmount1}" | bc` -eq 1 ]; then
    if [ "$(echo "${actTotal} > ${coinsTotalAmount1}" | bc)" -eq 1 ]; then
        echo "${name}     "
        exit 1
    else
        echo "${name}      "
    fi
}

function sendcoinsTx() {
    name=$1
    fromAdd=$2
    toAdd=$3
    note=$4
    amount=$5

    #
    #hash=$(sudo docker exec -it $1 ./dplatformos-cli coins transfer -t $2 -a $5 -n $4 | tr '\r' ' ')
    #echo $hash
    #sign=$(sudo docker exec -it $1 ./dplatformos-cli wallet sign -a $3 -d $hash | tr '\r' ' ')
    #echo $sign
    #sudo docker exec -it $1 ./dplatformos-cli wallet send -d $sign

    #
    #sudo docker exec -it $1 ./dplatformos-cli send coins transfer -a $5 -n $4 -t $2 -k $From
    result=$($name send coins transfer -a "${amount}" -n "${note}" -t "${toAdd}" -k "${fromAdd}" | tr '\r' ' ')
    echo "hash: $result"
    coinsgStr=$result
}

function genCoinsTx() {
    name=$1
    fromAdd=$2
    toAdd=$3
    note=$4
    amount=$5
    expire="600s"

    #
    hash=$(${name} coins transfer -t "${toAdd}" -a "${amount}" -n "${note}" | tr '\r' ' ')
    echo "${hash}"
    sign=$(${name} wallet sign -a "${fromAdd}" -d "${hash}" -e "${expire}" | tr '\r' ' ')
    echo "sign: $sign"
    coinsgStr=$sign

}

function sendCoinsTxHash() {
    name=$1
    sign=$2
    result=$(${name} wallet send -d "${sign}" | tr '\r' ' ')
    echo "hash: $result"
    coinsgStr=$result
}

function showCoinsBalance() {
    name=$1
    fromAdd=$2
    printf '==========showCoinBalance name=%s addr=%s==========\n' "${name}" "${fromAdd}"
    result=$($name account balance -e coins -a "${fromAdd}" | jq -r ".balance")
    printf 'balance %s \n' "${result}"
    coinsgStr=$result
}

function createAccount() {
    name=$1
    label=$2
    printf '==========CreateAccount name=%s ==========\n' "${name}"
    result=$($name account create -l "${label}" | jq -r ".acc.addr")
    printf 'New account addr %s \n' "${result}"
    coinsgStr=$result
}

function shouAccountList() {
    name=$1
    printf '==========shouAccountList name=%s ==========\n' "${name}"
    $name account list
}
