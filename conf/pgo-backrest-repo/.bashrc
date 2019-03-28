# .bashrc

awsKeySecret() {
    val=$(grep "$1" -m 1 /sshd/aws-s3-credentials.yaml | sed "s/^.*:\s*//")
    if [[ "$val" == "" ]]
    then
        echo "empty-$1"
    else
        echo "${val}"
    fi
}

export PGBACKREST_REPO1_S3_KEY=$(awsKeySecret "aws-s3-key")
export PGBACKREST_REPO1_S3_KEY_SECRET=$(awsKeySecret "aws-s3-key-secret")
