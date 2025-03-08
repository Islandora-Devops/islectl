FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y \
        openssh-server \
        curl \
        git \
        docker.io \
        docker-compose-v2 \
        sudo \
        libnss3-tools && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir /var/run/sshd && \
    sed -e 's|#PermitRootLogin|PermitRootLogin no\n\0|g' \
        -e 's|#PasswordAuthentication|PasswordAuthentication no\n\0|g' \
        /etc/ssh/sshd_config > /tmp/sshd && \
    mv /tmp/sshd /etc/ssh/sshd_config && \
    echo "HostKey /etc/ssh/ssh_host_ed25519_key" >> /etc/ssh/sshd_config
RUN useradd -ms /bin/bash foo && \
  echo 'foo:password' | chpasswd && \
  usermod -aG docker foo && \
  echo "foo ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/foo && \
  chmod 0440 /etc/sudoers.d/foo

RUN mkdir -p /home/foo/.ssh && chown foo:foo /home/foo/.ssh && chmod 700 /home/foo/.ssh
RUN curl -JLO "https://github.com/FiloSottile/mkcert/releases/download/v1.4.4/mkcert-v1.4.4-linux-amd64"
RUN mv mkcert-v1.4.4-linux-amd64 /usr/local/bin/mkcert && chmod +x /usr/local/bin/mkcert

COPY docker-entrypoint.sh /docker-entrypoint.sh

USER foo
RUN git config --global user.email "foo@localhost.localdomain" && \
    git config --global user.name "foo"

USER root
EXPOSE 22

ENTRYPOINT ["/docker-entrypoint.sh"]
