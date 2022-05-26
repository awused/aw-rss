FROM node:18
RUN apt-get update
RUN apt-get -y install rsync

WORKDIR /aw-rss

COPY package*.json ./

RUN npm install
RUN npm install -g npm-check-updates

COPY . .

