# Quatermain

![Quatermain](./assets/main.jpg)

Websites can be a maze, hiding deep inside pages that you probably ignored. 
Thanks to **quatermain** you will be able to explore every dark corner of your site.

## How it works?
Quatermain, leaving out the fictional description, is a tool that takes care of going through every link it finds on your site and generating a useful sitemap. 
This tool was born as a tool to be integrated into a deployment process or to be called via API in order to keep the sitemap of a website up to date. 
Operation is quite simple. Quatermain scans the first page that is passed to it as an argument, 
analyzes that it is a valid page by checking headers and meta tags, and for each valid link (without no follow, without no index, etc.) I repeat the procedure 

## Requirements
- `go v1.16`

## Install
`git clone git@github.com:CasvalDOT/quatermain.git`

`go mod vendor`

`go build`

`cp quatermain /usr/bin/`

## Usage

quatermain [OPTION]... [URL]

*Note! The URL must be finish with /*

```
quatermain https://mydomain.com/
```

```
quatermain -mc 200 -hb 120 https://mydomain.com/
```

The command provided start scan mydomain.com with a maximum pool of connections of 200 and stop when the script is inactive after 120 seconds
