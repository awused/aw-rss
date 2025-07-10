FROM node:24
RUN apt-get update
RUN apt-get -y install rsync vim

WORKDIR /aw-rss

COPY package*.json ./
COPY .bashrc /root/

RUN npm install --force
RUN npm install -g npm-check-updates

COPY . .

