# MsXpert

# Install (ubuntu console에서 실행)

## Web binary 준비
1. npm & node install
```
$ curl -sL https://deb.nodesource.com/setup_8.x | sudo -E bash -
$ sudo apt-get install -y nodejs
$ npm install -g @angular/cli@1.6.7

#eacces-permissions-errors
https://docs.npmjs.com/resolving-eacces-permissions-errors-when-installing-packages-globally#manually-change-npms-default-directory
```

2. node_module install
```
$ cd microservice/src/crossent/micro/studio/web
$ npm install
```

3. go-bindata install (go 설치되어 있다는 가정하에 실행)
```
# golang 1.9.7을 다운로드
$ wget https://dl.google.com/go/go1.9.7.linux-amd64.tar.gz
      
# /usr/local경로 아래에 golang 압축풀기
$ sudo tar -C /usr/local -xzf go1.9.7.linux-amd64.tar.gz
      
# ~/.profile 에서 PATH 등록하기
# export PATH=$PATH:/usr/local/go/bin
$ vi ~/.profile
$ source ~/.profile

$ cd microservice
$ export GOPATH=$PWD
$ export PATH=$PWD/bin:$PATH
$ go install vendor/github.com/jteeuwen/go-bindata/go-bindata
```

4. api 서버 주소 확인
```
microservice/src/crossent/micro/studio/web/src/environments/environment.prod.ts
# 예시
# msxpert portal에 할당하기 위해 생성해 놓은 public IP : 101.101.XXX.XXX
# IaaS영역에서 설정해둔 IP와 port번호를 이용하여 apiUrl, swaggerApiUrl 작성
      export const environment = {
        production: true,
        apiUrl: 'http://101.101.XXX.XXX:8080/api/v1',  
        swaggerApiUrl: 'http://101.101.XXX.XXX:8080/swagger/',
        cfEnvNameMSA: 'msa',
        cfEnvNamePrivate: 'private',
        msServices: 'config-server,registry-server,gateway-server',
        sampleApps: 'front,back',
        nodeTypeApp: 'App',
        nodeTypeService: 'Service',
        configService: 'config-server',
        registryService: 'registry-server',
        configServiceLabel: 'micro-config-server',
        registryServiceLabel: 'micro-registry-server'
      };

```

5. web binary 생성
```
$ cd microservice/src/crossent/micro/studio
$ make
```

## App 준비
1. config spring app 복사
```
$ cp config-0.0.1-SNAPSHOT.jar microservice/src/crossent/micro/broker/config/assets/configapp
```

2. registry spring app 복사
```
$ cp registry-0.0.1-SNAPSHOT.jar microservice/src/crossent/micro/broker/config/assets/registryapp
```

3. gateway spring app 복사
```
$ cp gateway-0.0.1-SNAPSHOT.jar microservice/src/crossent/micro/broker/config/assets/gatewayapp
```
## Blob 준비
```
$ cd microservice
$ bosh add-blob golang-linux-amd64-1.8.3.tar.gz golang/golang-linux-amd64-1.8.3.tar.gz
$ bosh add-blob jq-linux64-1.5 jq/jq-linux64-1.5
$ bosh add-blob grafana-5.3.4.linux-x64.tar.gz grafana/grafana-5.3.4.linux-x64.tar.gz
$ bosh add-blob nginx-1.10.3.tar.gz nginx/nginx-1.10.3.tar.gz
$ bosh add-blob pcre-8.42.tar.gz nginx/pcre-8.42.tar.gz
$ bosh add-blob prometheus-2.5.0.linux-amd64.tar.gz prometheus/prometheus-2.5.0.linux-amd64.tar.gz
$ bosh add-blob traefik-1.6.5_linux-amd64.gz traefik/traefik-1.6.5_linux-amd64.gz
```

## cf org, space 생성
```
$ cf create-org org-micro
$ cf create-space space-micro -o org-micro
```

## UAA Client 등록

UAA에 Microservice Client를 등록
```
# cf-uaac target 설정
## bosh-lite.com 부분은 상황에 맞게 Domain을 넣어줘야함.
$ uaac client add micro --name micro -s micro-secret \
   --authorities "oauth.login,scim.write,clients.read,scim.userids,password.write,clients.secret,clients.write,uaa.admin,scim.read,doppler.firehose" \
   --authorized_grant_types "authorization_code,client_credentials,password,refresh_token" \
   --scope "cloud_controller.read,cloud_controller.write,openid,cloud_controller.admin,scim.read,scim.write,doppler.firehose,uaa.user,routing.router_groups.read,uaa.admin,password.write" \
   --redirect_uri "https://uaa.bosh-lite.com/login"
```


## BOSH Deploy
```
$ cd microservice
$ bosh -e vbox update-cloud-config cloud-config.yml
$ bosh -e vbox create-release --name msxpert-nipa --force
$ bosh -e vbox upload-release --name msxpert-nipa
$ bosh -e vbox -d msxpert-nipa deploy microservice-msxpert.yml --vars-file vars-file.yml
$ bosh -e vbox -d msxpert-nipa run-errand broker-registrar

# vars-fil.yml 예시
 ---
       cf_api_url: https://api.bosh-lite.com # [cf api endpoint]
       uaa_url: https://uaa.bosh-lite.com
       cf_username: admin
       cf_password: # [CF_Admin_Password]
       cf_skip_cert_check: true
       external_url: # [studio-nipa VM과 연결시킬 Public IP]:[environment.prod.ts에 넣어준 port]
       haproxy_backend_port: 8089
       grafana_admin_password: admin 

```
