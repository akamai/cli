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
RUN mkdir /cli && \
    apk add --no-cache git python2 python2-dev py2-pip python3 python3-dev wget jq openssl openssl-dev  curl nodejs build-base libffi libffi-dev && \
    export AKAMAI_CLI_HOME=/cli && \
    wget `curl https://api.github.com/repos/akamai/cli/releases/latest | jq .assets[].browser_download_url | grep linuxamd64 | grep -v sig | sed s/\"//g` && \
    mv akamai-*-linuxamd64 /usr/local/bin/akamai && chmod +x /usr/local/bin/akamai && \
    mkdir -p /cli/.akamai-cli && \
    curl https://developer.akamai.com/cli/package-list.json | jq .packages[].name | sed s/\"//g | xargs akamai install --force

RUN echo "[cli]" > /cli/.akamai-cli/config && \
    echo "cache-path            = /cli/.akamai-cli/cache" >> /cli/.akamai-cli/config && \
    echo "config-version        = 1" >> /cli/.akamai-cli/config && \
    echo "enable-cli-statistics = false" >> /cli/.akamai-cli/config && \
    echo "last-ping             = 2018-04-27T18:16:12Z" >> /cli/.akamai-cli/config && \
    echo "client-id             =" >> /cli/.akamai-cli/config && \
    echo "install-in-path       =" >> /cli/.akamai-cli/config && \
    echo "last-upgrade-check    = 2018-04-27T22:19:59Z" >> /cli/.akamai-cli/config

ENV AKAMAI_CLI_HOME=/cli
VOLUME /root/.edgerc
VOLUME /cli
ENTRYPOINT ["/usr/local/bin/akamai"]
CMD ["--daemon"]
