# Add the following to your .bashrc, .bash_profile, or .zshrc, to make `akamai` work transparently on the host machine:
# function akamai {
#     if [[ `docker ps | grep akamai-cli$ | wc -l` -eq 1 ]]; then
#         docker exec -it akamai-cli akamai $@;
#     elif docker start akamai-cli > /dev/null 2>&1 && sleep 3 && docker exec -it akamai-cli akamai $@; then
#         return 0;
#     else
#         echo "Creating new docker container"
#         docker create -it -v $HOME/.edgerc:/root/.edgerc --name akamai-cli akamai/cli > /dev/null 2>&1 && akamai $@;
#     fi;
# }
# or, as a one-liner:
# function akamai { if [[ `docker ps | grep akamai-cli$ | wc -l` -eq 1 ]]; then docker exec -it akamai-cli akamai $@; elif docker start akamai-cli > /dev/null 2>&1 && sleep 3 && docker exec -it akamai-cli akamai $@; then return 0; else echo "Creating new docker container" docker create -it -v $HOME/.edgerc:/root/.edgerc --name akamai-cli akamai/cli > /dev/null 2>&1 && akamai $@; fi; }
FROM alpine 
ARG SOURCE_BRANCH=master
ARG AKAMAI_CLI_PACKAGE_REPO=https://developer.akamai.com/cli/package-list.json
ENV SOURCE_BRANCH="$SOURCE_BRANCH" GOROOT=/usr/lib/go GOPATH=/gopath GOBIN=/gopath/bin AKAMAI_CLI_HOME=/cli AKAMAI_CLI_PACKAGE_REPO="$AKAMAI_CLI_PACKAGE_REPO"
RUN mkdir -p /cli/.akamai-cli && \
    if [[ $SOURCE_BRANCH == "master" ]]; then \
        apk add --no-cache python2 python3 openssl nodejs libffi go && \
        apk add --no-cache -t .build-deps git python2-dev py2-pip python3-dev jq openssl-dev curl build-base libffi-dev npm && \
        export PATH=$PATH:$GOROOT/bin:$GOPATH/bin && \
        mkdir -p $GOBIN && \
        curl -s https://raw.githubusercontent.com/golang/dep/master/install.sh | sh && \
        go get github.com/akamai/cli && \
        cd $GOPATH/src/github.com/akamai/cli && \
        dep ensure && \
        go build -o /usr/local/bin/akamai; \
    else \
        apk add --no-cache python2 python3 openssl nodejs libffi go && \
        apk add --no-cache -t .build-deps git python2-dev py2-pip python3-dev jq openssl-dev curl build-base libffi-dev npm && \
        curl -s -o /usr/local/bin/akamai `curl https://api.github.com/repos/akamai/cli/releases/latest | jq -r .assets[].browser_download_url | grep linuxamd64 | grep -v sig`; \
    fi && \
    pip2 install --no-cache-dir --upgrade pip && \
    pip3 install --no-cache-dir --upgrade pip && \
    curl -s "$AKAMAI_CLI_PACKAGE_REPO" | jq -r .packages[].name | xargs akamai install --force && \
    apk del .build-deps

RUN echo "[cli]" > /cli/.akamai-cli/config && \
    echo "cache-path            = /cli/.akamai-cli/cache" >> /cli/.akamai-cli/config && \
    echo "config-version        = 1" >> /cli/.akamai-cli/config && \
    echo "enable-cli-statistics = false" >> /cli/.akamai-cli/config && \
    echo "last-ping             = $(date --utc +%FT%TZ)" >> /cli/.akamai-cli/config && \
    echo "client-id             =" >> /cli/.akamai-cli/config && \
    echo "install-in-path       =" >> /cli/.akamai-cli/config && \
    echo "last-upgrade-check    = ignore" >> /cli/.akamai-cli/config

VOLUME /root/.edgerc
VOLUME /cli
ENTRYPOINT ["/usr/local/bin/akamai"]
CMD ["--daemon"]
