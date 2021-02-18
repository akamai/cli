# Add the following to your .bashrc, .bash_profile, or .zshrc, to make `akamai` work transparently on the host machine:
# function akamai {
#     if [[ `docker ps | grep akamai-cli$ | wc -l` -eq 1 ]]; then
#         docker exec -it akamai-cli akamai $@;
#     elif docker start akamai-cli > /dev/null 2>&1 && sleep 3 && docker exec -it akamai-cli akamai $@; then
#         return 0;
#     else
#         echo "Creating new docker container"
#	  mkdir -p $HOME/.akamai-cli-docker
#         docker create -it -v $HOME/.edgerc:/root/.edgerc -v $HOME/.akamai-cli-docker:/cli --name akamai-cli akamai/cli > /dev/null 2>&1 && akamai $@;
#     fi;
# }
# or, as a one-liner:
# function akamai { if [[ `docker ps | grep akamai-cli$ | wc -l` -eq 1 ]]; then docker exec -it akamai-cli akamai $@; elif docker start akamai-cli > /dev/null 2>&1 && sleep 3 && docker exec -it akamai-cli akamai $@; then return 0; else echo "Creating new docker container" && mkdir -p $HOME/.akamai-cli-docker && docker create -it -v $HOME/.edgerc:/root/.edgerc -v $HOME/.akamai-cli-docker:/cli --name akamai-cli akamai/cli > /dev/null 2>&1 && akamai $@; fi; }
FROM golang:1.14
ENV AKAMAI_CLI_HOME=/cli
RUN /bin/bash -c 'mkdir /cli && \
    apt update && \
    apt install -y python-pip python3 python3-pip jq libssl-dev npm && \
    pip2 install --upgrade pip && \
    pip3 install --upgrade pip && \
    go get github.com/akamai/cli && \
    cd $GOPATH/src/github.com/akamai/cli && \
    go mod tidy && \
    go build -o akamai-master-linuxamd64 cli/main.go; \
    mv akamai-*-linuxamd64 /usr/local/bin/akamai && chmod +x /usr/local/bin/akamai && \
    mkdir -p /cli/.akamai-cli && \
    curl -A "" https://developer.akamai.com/cli/package-list.json | jq .packages[].name | sed s/\"//g | xargs akamai install --force'

RUN echo "[cli]" > /cli/.akamai-cli/config && \
    echo "cache-path            = /cli/.akamai-cli/cache" >> /cli/.akamai-cli/config && \
    echo "config-version        = 1" >> /cli/.akamai-cli/config && \
    echo "enable-cli-statistics = false" >> /cli/.akamai-cli/config && \
    echo "last-ping             = 2018-04-27T18:16:12Z" >> /cli/.akamai-cli/config && \
    echo "client-id             =" >> /cli/.akamai-cli/config && \
    echo "install-in-path       =" >> /cli/.akamai-cli/config && \
    echo "last-upgrade-check    = ignore" >> /cli/.akamai-cli/config

VOLUME /root/.edgerc
VOLUME /cli
ENTRYPOINT ["/usr/local/bin/akamai"]
CMD ["--daemon"]
