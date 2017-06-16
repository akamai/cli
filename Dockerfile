FROM ubuntu
MAINTAINER Kirsten Hunter (khunter@akamai.com), Davey Shafik (dshafik@akamai.com)
RUN apt-get update
RUN apt-get install -y git curl patch gawk g++ gcc make libc6-dev patch libreadline6-dev zlib1g-dev libssl-dev libyaml-dev libsqlite3-dev sqlite3 autoconf libgdbm-dev libncurses5-dev automake libtool bison pkg-config libffi-dev
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y -q libssl-dev python-all wget vim python-pip php7.0 ruby-dev nodejs-dev npm ruby perl golang-go
RUN pip install httpie-edgegrid 
ADD . /opt
WORKDIR /opt
RUN mkdir bin
RUN export PATH=${PATH}:/opt/bin
RUN export GOPATH=/opt/bin
RUN curl https://glide.sh/get | sh
RUN echo "export PATH=${PATH}:/opt/bin" >> /root/.bashrc
RUN echo "export GOPATH=/opt/bin" >> /root/.bashrc
RUN echo "PS1='Akamai CLI Sandbox >> '" >> /root/.bashrc
ENTRYPOINT ["/bin/bash"]
