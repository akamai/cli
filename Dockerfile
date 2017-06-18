FROM ubuntu
MAINTAINER Kirsten Hunter (khunter@akamai.com), Davey Shafik (dshafik@akamai.com)
RUN apt-get update
RUN apt-get install -y git curl patch gawk g++ gcc make libc6-dev patch libreadline6-dev zlib1g-dev libssl-dev libyaml-dev libsqlite3-dev sqlite3 autoconf libgdbm-dev libncurses5-dev automake libtool bison pkg-config libffi-dev software-properties-common
RUN add-apt-repository -y ppa:longsleep/golang-backports
RUN apt-get update 
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y -q libssl-dev python-all wget vim python-pip php7.0 ruby-dev ruby perl golang-go 
RUN pip install httpie-edgegrid 
ADD . /opt/src/github.com/akamai/cli
WORKDIR /opt
RUN curl -sL https://deb.nodesource.com/setup_7.x | sh -
RUN apt-get install nodejs
RUN mkdir bin
RUN export PATH=${PATH}:/opt/bin
RUN ln -s /usr/bin/nodejs /usr/bin/node
RUN export GOPATH=/opt
ENV GOPATH=/opt
ENV PATH=${PATH}:/opt/bin
RUN curl https://glide.sh/get | sh
WORKDIR /opt/src/github.com/akamai/cli
RUN glide install
RUN go build -o akamai . && mv akamai /opt/bin/akamai
RUN echo "export PATH=${PATH}:/opt/bin" >> /root/.bashrc
RUN echo "export GOPATH=/opt" >> /root/.bashrc
RUN echo "PS1='Akamai CLI Sandbox >> '" >> /root/.bashrc
WORKDIR /opt
RUN akamai get akamai/cli-property
RUN akamai get akamai/cli-purge
ENTRYPOINT ["/bin/bash"]
