import { Component, OnInit } from '@angular/core';
import { DomSanitizer, SafeResourceUrl } from '@angular/platform-browser';
import { ActivatedRoute, Router  } from "@angular/router";
import { environment } from '../../../environments/environment';
import { MicroApi } from '../models/microapi.model';
import { Part } from '../models/part.model';
import { ApiService } from '../../shared/api.service'
import { PartService } from '../api-create/part.service';

@Component({
  selector: 'app-api-view',
  templateUrl: './api-view.component.html',
  styleUrls: ['./api-view.component.css']
})
export class ApiViewComponent implements OnInit {
  apiUrl: string = 'apigateway';
  id: string;
  microapi: MicroApi = new MicroApi();
  swaggerApiUrl: string = environment.swaggerApiUrl;
  api: string = environment.apiUrl;
  iframeSrc: SafeResourceUrl;
  isIframe: boolean = false;

  constructor(private apiService: ApiService,
              private sanitizer: DomSanitizer,
              private route: ActivatedRoute,
              private partService: PartService) { }

  ngOnInit() {
    this.id = this.route.snapshot.params['id'];
    this.iframeSrc = this.sanitizer.bypassSecurityTrustResourceUrl('about:blank');
    this.getMicroapi();
  }

  getMicroapi() {
    this.apiService.get<MicroApi>(`${this.apiUrl}/${this.id}`).subscribe(
      data => {
        console.log(data)
        this.microapi = data;
      }
    );
  }

  getSwagger() {
    if (this.isIframe == false && this.microapi) {
      console.log(this.swaggerApiUrl);
      this.iframeSrc = this.sanitizer.bypassSecurityTrustResourceUrl(this.swaggerApiUrl + '/entry/?id=' + this.microapi.microId + '&domain=' + this.api);
    }
    this.isIframe = !this.isIframe;
  }
}
