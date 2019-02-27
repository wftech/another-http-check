FROM golang:1.11-alpine

ARG APP_GID
ARG APP_USER

RUN apk add gcc g++ ca-certificates git curl vim

RUN adduser -D -u ${APP_GID} -g ${APP_USER} ${APP_USER}

USER ${APP_USER}

RUN curl -fLo ~/.vim/autoload/plug.vim --create-dirs \
        https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim

RUN echo "call plug#begin('~/.vim/plugged')" > ~/.vimrc
RUN echo "Plug 'fatih/vim-go', { 'do': ':GoUpdateBinaries' }" >> ~/.vimrc
RUN echo "call plug#end()" >> ~/.vimrc
RUN echo "set tabstop=4" >> ~/.vimrc
RUN echo "set expandtab" >> ~/.vimrc
RUN echo "set shiftwidth=4" >> ~/.vimrc
RUN echo "set t_Co=256" >> ~/.vimrc
RUN echo "set number" >> ~/.vimrc

RUN vim -E -s -u ~/.vimrc +PlugInstall +qall

ENV GO111MODULE=on

WORKDIR /app

