import { Component, OnInit } from '@angular/core';
import { DomSanitizer, SafeResourceUrl } from '@angular/platform-browser';
import { ActivatedRoute, Router  } from "@angular/router";
import { NgForm } from '@angular/forms';
import { MicroApi } from '../models/microapi.model';
import { Part } from '../models/part.model';
import { ApiService } from '../../shared/api.service'
import { PartService } from './part.service';
import { Micro } from '../../view/micro-list/micro.model';
import { KeyValue } from '../models/KeyValue.model';
import { environment } from '../../../environments/environment';
import { forkJoin } from "rxjs/observable/forkJoin";
import { Org } from '../../shared/models/org.model';
import { LoaderService } from '../../shared/loader.service'

@Component({
  selector: 'app-api-create',
  templateUrl: './api-create.component.html',
  styleUrls: ['./api-create.component.css']
})
export class ApiCreateComponent implements OnInit {
  apiUrl: string = 'apigateway';
  offset: number = 0;
  microapi: MicroApi = new MicroApi();
  parts: Part[];
  methods = [{name:'', value:''}, {name:'GET', value:'get'}, {name:'POST', value:'post'}, {name:'PUT', value:'put'}];
  pathstrips = ['N', 'Y'];
  micros: Micro[] = [];
  micro: Micro = new Micro(0, "", "", "", "");
  headerKey: string;
  headerVal: string;
  swaggerApiUrl: string = environment.swaggerApiUrl;
  api: string = environment.apiUrl;
  iframeSrc: SafeResourceUrl;
  isIframe: boolean = false;
  id: string;
  isId: boolean = false;
  isNameValid: boolean = true;
  isPathValid: boolean = true;
  orgs: Org[];
  isFirst: boolean = true;
  isSecond: boolean = false;

  constructor(private apiService: ApiService,
              private loaderService: LoaderService,
              private router: Router,
              private sanitizer: DomSanitizer,
              private route: ActivatedRoute,
              private partService: PartService) { }

  ngOnInit() {
    this.microapi.host = '';
    this.microapi.pathStrip = 'Y';
    this.microapi.maxconn = '10';
    this.microapi.period = '3';
    this.microapi.average = '5';
    this.microapi.burst = '10';
    this.microapi.headers = [];
    this.parts = this.partService.partsList();
    this.iframeSrc = this.sanitizer.bypassSecurityTrustResourceUrl('about:blank');
    this.id = this.route.snapshot.params['id'];

    if(this.id){
      this.isId = true;
      this.getMicroapi()
    } else {
      this.listOrgs();
      this.listFrontends();
    }
  }

  getMicroapi() {
    let listFrontends = this.apiService.get<Micro[]>(`${this.apiUrl}/frontend/microservices`);
    let getMicroapi = this.apiService.get<MicroApi>(`${this.apiUrl}/${this.id}/rule`);
    let listOrgs = this.apiService.get<Org[]>('orgs');

    forkJoin([listFrontends, getMicroapi, listOrgs]).subscribe(results => {
        this.micros = results[0];
        this.microapi = results[1];
        this.orgs = results[2]['resources'];
        this.micro = this.micros.filter(m => m.id == this.microapi.microId)[0];
      },
      err => {
        this.loaderService.forceHide();
        setTimeout(() => {
          //alert('no auth');
          this.router.navigate(['apimanage']);
        }, 100)

      }
    );

    // this.apiService.get<MicroApi>(`${this.apiUrl}/${this.id}/rule`).subscribe(
    //   data => {
    //     console.log(data)
    //     this.microapi = data;
    //     this.micro = this.micros.filter(m => m.id == this.microapi.microId);
    //     console.log(this.micro)
    //   }
    // );
  }

  listFrontends() {
    this.apiService.get<Micro[]>(`${this.apiUrl}/frontend/microservices`).subscribe(
      data => {
        console.log(data)
        this.micros = data;
      }
    );
  }

  listOrgs() {
    this.apiService.get('manageorgs').subscribe(
      data => {
        this.orgs = data['resources'];
        // if(this.orgs.length > 0) {
        //   this.org = this.orgs[0];
        //this.ComposeRequest.orgGuid = this.orgs[0].metadata.guid;
        //this.ComposeRequest.orgName = this.orgs[0].entity.name;
        // }
      }
    );
  }

  onChangeMicro() {
    this.apiService.post(`${this.apiUrl}/${this.micro.id}/swagger`, {}).subscribe(
      data => {
        console.log(data);
      }
    );
  }

  plus() {
    this.microapi.headers.push(new KeyValue(this.headerKey, this.headerVal));
    this.headerKey = "";
    this.headerVal = "";
  }

  minus(index) {
    this.microapi.headers.splice(index, 1);
  }

  save(){
    console.log(this.microapi)
    if(!confirm("저장하시겠습니까?")) {
      return;
    }

    this.microapi.microId = this.micro.id;
    this.microapi.image = Math.floor((Math.random()*250)).toString() + ',' + Math.floor((Math.random()*250)).toString() + ',' + Math.floor((Math.random()*250)).toString();

    this.apiService.post<MicroApi>(`${this.apiUrl}`, this.microapi).subscribe(
      data => {
        this.router.navigate(['/apimanage']);
      }
    );
  }

  getSwagger() {
    if (this.isIframe == false && this.micro) {
      this.iframeSrc = this.sanitizer.bypassSecurityTrustResourceUrl(this.swaggerApiUrl + '/entry/?id=' + this.micro.id + '&domain=' + this.api);
    }
    this.isIframe = !this.isIframe;
  }

  Next(form : NgForm) {
    // let result = <any>{};
    if(form.controls['name'].valid == true && form.controls['name'].valid != false){
      this.apiService.get<any>(`${this.apiUrl}/name/check?name=`+this.microapi.name).subscribe(
        data => {
          if(data.result == 'ok') {
            this.isNameValid = true
          }else{
            this.isNameValid = false
          }
        }
      );
    }
  }

  PathNext(form : NgForm) {
    if(form.controls['path'].valid == true && form.controls['path'].valid != false){
      this.apiService.get<any>(`${this.apiUrl}/path/check?path=`+this.microapi.path).subscribe(
        data => {
          if(data.result == 'ok') {
            this.isPathValid = true
          }else{
            this.isPathValid = false
          }
        }
      );
    }
  }

  nextApi(next: boolean) {
    if(next) {
      this.isFirst = false;
      this.isSecond = true;
    } else {
      this.isFirst = true;
      this.isSecond = false;
    }
  }
}
