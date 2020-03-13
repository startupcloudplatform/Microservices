import { Component, OnInit } from '@angular/core';
import { Micro } from './micro.model'
import { Circle } from './circle.model'
import { Link } from './link.model'

import { ApiService } from '../../shared/api.service'


@Component({
  selector: 'app-micro-list',
  templateUrl: './micro-list.component.html',
  styleUrls: ['./micro-list.component.css']
})
export class MicroListComponent implements OnInit {
  apiUrl: string = 'microservices?offset=';
  offset: number = 0;
  micros: Micro[];
  isEnd: boolean = false;
  text: string = "";
  searchName: string = "";

  constructor(private apiService: ApiService) { }

  ngOnInit() {
    this.listMicroservice();
  }


  listMicroservice() {
    this.apiService.get<Micro[]>(`${this.apiUrl}${this.offset}`).subscribe(
      // data => this.micros = [{id: data['id'], orgName: data['orgName'], spaceName: data['spaceName'], name: data['name'], version: data['version'], desc: data['description'], app: data['app'], service: data['space'], status: data['status']}]
    //data => this.micros = [{orgName: data['orgName'], spaceName: data['spaceName'], name: data['name']}]

      data => {
        this.micros = data;

        // for(let i=0;i<this.micros.length;i++){
        //   this.micros[i].circles = this.createRange(this.micros[i].app);
        // }
        // for(let i=0;i<this.micros.length;i++){
        //   this.micros[i].links = this.createLink(this.micros[i].circles);
        // }
        this.setView(this.micros);
      }

      //data => this.micros.push(data)


    );
  }

  more() {
    this.offset += 6;
    this.apiService.get<Micro[]>(`${this.apiUrl}${this.offset}`).subscribe(
      data => {
        data.forEach(d => {this.micros.push(d)});

        this.setView(this.micros);

        if(this.micros.length <= this.offset){
          this.isEnd = true;
        }
      }
    );
  }

  setView(data){
    for(let i=0;i<data.length;i++){
      if( !data[i].circles ){
        data[i].circles = this.createRange(this.micros[i].app);
      }
    }
    for(let i=0;i<data.length;i++){
      if( !data[i].links ){
        data[i].links = this.createLink(this.micros[i].circles);
      }
    }
  }

  createRange(number){
    let items: Circle[] = [];
    let n = 255-((number/10)*255);
    for(let i = 0; i < number; i++){
      let x = Math.floor(Math.random()*330);
      let y = Math.floor(Math.random()*140)+10;
      let radius = Math.floor(Math.random()*20);

      let r = Math.floor(n);
      let g = Math.floor(n);
      let b = Math.floor(n);

      let c = new Circle(x, y, radius, "rgba(" + r + "," + g + "," + b + ",1)");

      items.push(c);
    }
    return items;
  }

  createLink(circles: Circle[]){
    let items: Link[] = [];
    for(let i = 0; i < circles.length; i++){
      let x1 = circles[i].x;
      let y1 = circles[i].y;

      for(let j=0; j<circles.length; j++){
        if (i != j) {
          let c = new Link(x1, y1, circles[j].x, circles[j].y);

          items.push(c);
        }
      }
    }
    return items;
  }

}


