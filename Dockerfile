FROM node:14
RUN apt-get update
RUN apt-get -y install rsync

WORKDIR /aw-rss

COPY package*.json ./

RUN npm install

COPY . .

