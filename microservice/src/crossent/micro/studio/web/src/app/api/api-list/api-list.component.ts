import { Component, OnInit, Input, Output, EventEmitter } from '@angular/core';
import { DomSanitizer, SafeResourceUrl } from '@angular/platform-browser';
import { environment } from '../../../environments/environment';
import { MicroApi } from '../models/microapi.model';
import { ApiService } from '../../shared/api.service';
import { PartService } from '../api-create/part.service';
import { Part } from '../models/part.model';

declare const $: any;

@Component({
  selector: 'api-list',
  templateUrl: './api-list.component.html',
  styleUrls: ['./api-list.component.css']
})
export class ApiListComponent implements OnInit {
  @Input() microId: number = 0;
  @Output() onSaveApi = new EventEmitter<MicroApi>();

  apiUrl: string = 'apigateway?offset=';
  offset: number = 0;
  microapis: MicroApi[];
  parts: Part[];
  searchName: string = "";
  searchPath: string = "";
  isEnd: boolean = false;
  part: string;
  isList: boolean = false;
  iframeSrc: SafeResourceUrl;
  swaggerApiUrl: string = environment.swaggerApiUrl;
  api: string = environment.apiUrl;
  swaggerApiName: string = "";
  selectedMicroapi: MicroApi;

  constructor(private apiService: ApiService,
              private sanitizer: DomSanitizer,
              private partService: PartService) { }

  ngOnInit() {
    this.parts = this.partService.partsList();
    this.listApis();
    if (this.microId > 0) {
      this.isList = true;
    }
    this.iframeSrc = this.sanitizer.bypassSecurityTrustResourceUrl('about:blank');
    this.selectedMicroapi = new MicroApi();

  }

  listApis() {
    this.apiService.get<MicroApi[]>(`${this.apiUrl}${this.offset}`).subscribe(
      data => {
        this.microapis = data;
      }
    );
  }

  more() {
    this.offset += 6;
    this.apiService.get<MicroApi[]>(`${this.apiUrl}${this.offset}`).subscribe(
      data => {
        data.forEach(d => {this.microapis.push(d)});
        if(this.microapis.length <= this.offset){
          this.isEnd = true;
        }
      }
    );
  }

  getSwagger(microApi: MicroApi) {
    this.swaggerApiName = microApi.name;
    $('.shape').shape('flip right');
    if (this.microId != 0) {
      this.apiService.get<MicroApi>(`apigateway/${microApi.id}`).subscribe(
        data => {
          this.iframeSrc = this.sanitizer.bypassSecurityTrustResourceUrl(this.swaggerApiUrl + '/entry/?id=' + data.microId + '&domain=' + this.api);
        }
      );
    }
  }

  goPrev(){
    this.swaggerApiName = "";
    $('.shape').shape('flip right');
  }

  showAddApi(microApi: MicroApi){
    $('.ui.modal.addapi')
      .modal({
        inverted: true
      })
      .modal('show')
    ;
    this.selectedMicroapi.username = "";
    this.selectedMicroapi.userpassword = "";
    this.selectedMicroapi = microApi;
  }


  addApi(){
    console.log("-- api-list.component.html --")
    if (this.microId != 0 && this.selectedMicroapi.username != "" && this.selectedMicroapi.userpassword != "") {
      this.selectedMicroapi.microId = this.microId;

      this.apiService.post<MicroApi>(`apigateway/${this.microId}/api`, this.selectedMicroapi).subscribe(
        data => {
          this.onSaveApi.emit(this.selectedMicroapi);
        },
        err => {
          alert(err.error);
        }
      );
    }

  }

}
