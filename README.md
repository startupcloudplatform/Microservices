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
$ cd microservice
$ export GOPATH=$PWD
$ export PATH=$PWD/bin:$PATH
$ go install vendor/github.com/jteeuwen/go-bindata/go-bindata
```

4. api 서버 주소 확인
```
microservice/src/crossent/micro/studio/web/src/environments/environment.prod.ts
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

## cf org, space 생성
```
$ cf create-org org-micro
$ cf create-space space-micro -o org-micro
```

## UAA Client 등록

UAA에 Microservice Client를 등록
```
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
```
