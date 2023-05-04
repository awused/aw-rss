FROM node:20
RUN apt-get update
RUN apt-get -y install rsync

WORKDIR /aw-rss

COPY package*.json ./

RUN npm install --force
RUN npm install -g npm-check-updates

COPY . .

