#!/usr/bin/env bash

export POD_NAME=""
function cleanup() {
    [[ -n "${POD_NAME}" ]] && kubectl delete pod ${POD_NAME} >/dev/null 2>&1 || true
}
trap cleanup EXIT TERM

function _list_appdb() {
    NS_ARG="--all-namespaces"
    [[ -n "$1" ]] && NS_ARG="-n ${1}"
    
    IFS=';' read -ra items <<< "$(kubectl get appdb $NS_ARG -o go-template='{{range .items}}{{.metadata.name}}:{{.metadata.namespace}}:{{.spec.appDBInstance}}:{{.spec.dbName}}:{{index .spec.users 0}}:{{.status.provisioning}}{{"\n"}}{{end}}' | sort -k 2 -k 1 -t: | tr '\n' ';')"
    local count=1
    lines=$(for i in ${items[@]}; do
        if [[ $count -eq 1 ]]; then
            printf "num\tname\tnamespace\tappdbi\tdbname\tuser\tprovisioning\n"
        fi
        IFS=":" read -ra TOKS <<< "${i}"
        printf "$count)\t${TOKS[0]}\t${TOKS[1]}\t${TOKS[2]}\t${TOKS[3]}\t${TOKS[4]}\t${TOKS[5]}\n"
        ((count=count+1))
    done | column -t)
    count=$(echo "$lines" | wc -l)
    echo "$lines" >&2
    local sel=0
    while [[ $sel -lt 1 || $sel -gt $count ]]; do
        read -p "Select an ApDB: " sel >&2
    done
    echo "${items[(sel-1)]}"
}

function _make_appdb_shell_podspec() {
    local secretName=$1
    local user=$2
    local dbname=$3
    read -r -d '' SPEC_JSON <<EOF
{
  "apiVersion": "v1",
  "spec": {
    "containers": [{
      "name": "appdb-shell",
      "image": "arey/mysql-client",
      "command": ["mysql", "-u", "${user}", "${dbname}"],
      "env": [
        {
          "name": "MYSQL_HOST",
          "valueFrom": {
            "secretKeyRef": {
              "key": "dbhost",
              "name": "${secretName}"
            }
          }
        },
        {
          "name": "MYSQL_PWD",
          "valueFrom": {
            "secretKeyRef": {
              "key": "password",
              "name": "${secretName}"
            }
          }
        }
      ],
      "stdin": true,
      "stdinOnce": true,
      "tty": true
    }]
  }
}
EOF
    echo "${SPEC_JSON}"
}

function kube-appdb-shell() {
    local appdb=$1
    local namespace=$2

    local secretNameDBUser=$(kubectl -n $namespace get appdb $appdb -o go-template='{{index .status.credentialsSecrets (index .spec.users 0)}}:{{.spec.dbName}}:{{index .spec.users 0}}')
    IFS=":" read -ra TOKS <<< "${secretNameDBUser}"
    local secretName=${TOKS[0]}
    local dbname=${TOKS[1]}
    local user=${TOKS[2]}
    SPEC_JSON=$(_make_appdb_shell_podspec $secretName $user $dbname)
    id=$(printf "%x" $((RANDOM + 100000)))
    POD_NAME="appdb-shell-${id}"
    kubectl run -n ${namespace} ${POD_NAME} -i -t --rm --restart=Never --image=arey/mysql-client --overrides="${SPEC_JSON}"
}

appdb=$1
namespace=${KUBECTL_PLUGINS_CURRENT_NAMESPACE:-default}
if [[ -z "${appdb}" ]]; then
    SEL=$(_list_appdb)
    IFS=":" read -ra APPDB <<< "${SEL}"
    appdb=${APPDB[0]}
    namespace=${APPDB[1]}
fi

kube-appdb-shell $appdb $namespace