import {Component, OnInit} from '@angular/core';
import {MicroApi} from "../models/microapi.model";
import {environment} from "../../../environments/environment";
import {DomSanitizer, SafeResourceUrl} from "@angular/platform-browser";
import {ApiService} from "../../shared/api.service";
import {ActivatedRoute} from "@angular/router";
import {PartService} from "../api-create/part.service";
import {MicroApiComponent} from "../../view/micro-api/micro-api.component";

@Component({
  selector: 'api-user-list',
  templateUrl: './api-user-list.component.html',
  styleUrls: ['./api-user-list.component.css']
})
export class ApiUserListComponent implements OnInit{
  apiUrl: string = 'apigateway';
  id: string;
  microapis: MicroApi[] = [];
  apiName: string;
  apiPath: string;
  //swaggerApiUrl: string = environment.swaggerApiUrl;
  api: string = environment.apiUrl;
  iframeSrc: SafeResourceUrl;
  //isIframe: boolean = false;

  constructor(private apiService: ApiService,
              private sanitizer: DomSanitizer,
              private route: ActivatedRoute,
              ) { }

  ngOnInit() {
    this.id = this.route.snapshot.params['id'];
    this.iframeSrc = this.sanitizer.bypassSecurityTrustResourceUrl('about:blank');
    this.getMicroApiUserList();
    //this.getMicroApiUserList();
    console.log(this.id)
  }

  getMicroApiUserList() {
    this.apiService.get<MicroApi[]>(`${this.apiUrl}/${this.id}/testb`).subscribe(
      data => {
        this.apiName = data[0].name;
        this.apiPath = data[0].path;
        this.microapis = data;
      }
    );
  }
  /*
  getMicroApiUserList() {
    this.apiService.get<String[]>(`${this.apiUrl}/${this.id}`).subscribe(
      data => {
        console.log(data)
        //this.microapi = data;
      }
    );
  }
  */

}
