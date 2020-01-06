declare var jQuery: any;

import { Component, OnInit, ViewChild, ElementRef } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { DomSanitizer, SafeResourceUrl } from '@angular/platform-browser';

import { ApiService } from '../../shared/api.service';
import { Microservice } from '../../shared/models/microservice.model';
import { AppDroppableDirective } from '../../d3-studio/directives/app-droppable.directive';
import { ZoomableDirective } from '../../d3-studio/directives/zoomable.directive';
import { Node } from '../../d3-studio/shared/node.model';
import { Link } from '../../d3-studio/shared/link.model';
import { environment } from '../../../environments/environment';
import { MicroApi } from '../../api/models/microapi.model';

@Component({
  selector: 'app-edit',
  templateUrl: './edit.component.html',
  styleUrls: ['./edit.component.css']
})
export class EditComponent implements OnInit {
  @ViewChild(AppDroppableDirective) appDropDirective = null;
  @ViewChild(ZoomableDirective) directive = null;
  @ViewChild('tabNetwork') tabNetwork: ElementRef;
  @ViewChild('tabRouting') tabRouting: ElementRef;

  apiUrl: string = 'microservices';
  micro = new Microservice(0,null,null,'','','','false','');
  public accordion = false;
  marketApps = [];
  marketServices = [];
  Apis = [];
  gatewayApp: any;
  configService: any;
  registryService: any;
  networkPolicyMaps = new Map();
  viewNetwork: any;

  searchAppName: string;
  searchServiceName: string;
  searchApiName: string;

  nodes: Node[] = [];
  links: Link[] = [];
  routes: any[] = [];//[{linkId: '', service: '', path: ''}];
  bindings: {app:string; service:string}[] = [];
  configs: {app:string; property:string}[] = [];

  nodeDatas: Map<string, any> = new Map();
  modal: any = {};

  delNodes: Node[] = [];

  microapis: MicroApi[];
  selectedMicroapi: MicroApi;
  iframeSrc: SafeResourceUrl;
  swaggerApiUrl: string = environment.swaggerApiUrl;
  api: string = environment.apiUrl;

  constructor(private apiService: ApiService,
              private route: ActivatedRoute,
              private _el : ElementRef,
              private router: Router,
              private sanitizer: DomSanitizer) {
    this.micro.id = route.snapshot.params['id'];
  }

  ngOnInit() {
    jQuery('.menu .item').tab();

    this.getMicroservice();
    this.getMicroserviceDetail();
    this.initViewNetwork();

    this.listAppApis();
    this.selectedMicroapi = new MicroApi();
    this.iframeSrc = this.sanitizer.bypassSecurityTrustResourceUrl('about:blank');
  }


  getMicroservice() {
    this.apiService.get<Microservice>(`${this.apiUrl}/${this.micro.id}`).subscribe(
      data => {
        this.micro = data['microservice'];
      }
    );
  }

  getMicroserviceDetail() {
    let nodeMap = new Map();
    this.apiService.get<any>(`${this.apiUrl}/${this.micro.id}/composition`).subscribe(
      data => {
        var dragLineId = 1000;

        let composeApps = data['apps'];
        let composeServices = data['services'];
        let composeBindings = data['bindings'];
        let composeRoutes = data['routes'];
        let composePolicies = data['policies'];
        let composeProperties = data['properties'];

        let svg = this._el.nativeElement.querySelector('svg');
        let _x = svg.width.baseVal.value;
        let _y = svg.height.baseVal.value;
        let _distance = 100;
        let xVal = [];
        let yVal = [];

        // Do not display config-server and registry-server
        let configServiceIndex = 0, registryServiceIndex = 0;
        for(var i = 0; i < composeServices.length; i++) {
          if(composeServices[i].entity.name.indexOf(environment.configService) >= 0) {
            configServiceIndex = i;
            this.configService = composeServices[i];
          } else if(composeServices[i].entity.name.indexOf(environment.registryService) >= 0) {
            registryServiceIndex = i;
            this.registryService = composeServices[i];
          }
        }
        composeServices.splice(configServiceIndex,1);
        if(configServiceIndex < registryServiceIndex) registryServiceIndex--;
        composeServices.splice(registryServiceIndex,1);
        let nodesCnt = composeServices.length + composeApps.length;
        let posCnt = -1;

        // services
        for(var i = 0; i < composeServices.length; i++) {
          posCnt++;
          xVal[posCnt] = (_x / 2 + _distance * Math.cos(2 * Math.PI * posCnt / nodesCnt));
          yVal[posCnt] = (_y / 2 + _distance * Math.sin(2 * Math.PI * posCnt / nodesCnt));
          let _node = {
            shape: "circle",
            x: xVal[posCnt],
            y: yVal[posCnt],
            r: 25,
            id: composeServices[i].metadata.guid,
            name: composeServices[i].entity.name,
            type: "Service",
            color: "rgb(177,130,186)"
          };
          nodeMap.set(composeServices[i].metadata.guid, _node);
          this.nodeDatas.set(composeServices[i].metadata.guid, composeServices[i]);
          this.putNodes(_node);
        }

        // apps
        for(var i = 0; i < composeApps.length; i++) {
          posCnt++;
          xVal[posCnt] = (_x / 2 + _distance * Math.cos(2 * Math.PI * posCnt / nodesCnt));
          yVal[posCnt] = (_y / 2 + _distance * Math.sin(2 * Math.PI * posCnt / nodesCnt));
          let _node = {
            shape: "circle",
            x: xVal[posCnt],
            y: yVal[posCnt],
            r: 25,
            id: composeApps[i].metadata.guid,
            name: composeApps[i].entity.name,
            type: "App",
            color: "rgb(155,208,198)"
          };
          nodeMap.set(composeApps[i].metadata.guid, _node);
          this.nodeDatas.set(composeApps[i].metadata.guid, composeApps[i]);
          this.putNodes(_node);
          if (composeApps[i].entity.name.startsWith("gatewayapp")) {
            this.gatewayApp = composeApps[i];
          }
        }

        // app & service bindings
        composeBindings.forEach(bind => {
          if(bind.service_instance_guid == this.configService.metadata.guid || bind.service_instance_guid == this.registryService.metadata.guid) {
            return;
          }
          dragLineId++;
          let source = nodeMap.get(bind.app_guid);
          let target = nodeMap.get(bind.service_instance_guid);
          var link = {
            id: dragLineId,
            type: environment.nodeTypeService,
            sNode: {id: source.id, x: source.x, y: source.y},
            tNode: {id: target.id, x: target.x, y: target.y},
            source: source.id,
            target: target.id
          };
          this.links.push(<Link>link);
          this.putLink(link);
          setTimeout(() => {
            jQuery('#link-path-'+link.id).addClass('service');
          });
        });

        // network-policies
        composePolicies.forEach(policy => {
          dragLineId++;
          let source = nodeMap.get(policy.source.id);
          let target = nodeMap.get(policy.destination.id);
          let type = environment.nodeTypeApp;

          if(source && target) {
            if (target.type == environment.nodeTypeService) {
              type = environment.nodeTypeService
            }
            var link = {
              id: dragLineId,
              type: type,
              sNode: {id: source.id, x: source.x, y: source.y},
              tNode: {id: target.id, x: target.x, y: target.y},
              source: source.id,
              target: target.id
            };
            this.links.push(<Link>link);
            if (source.type == environment.nodeTypeApp && target.type == environment.nodeTypeService) {
              link.type = environment.nodeTypeService;
              if (!source.name.startsWith("gatewayapp")) {
                this.bindings.push({app: source.id, service: target.id});
              }
            }
            // app <-> gatewayApp : ass network policy
            if (source.type == environment.nodeTypeApp && target.type == environment.nodeTypeApp) {
              this.viewNetwork = {
                id: link.id,
                source: source,
                target: target,
                protocol: 'tcp',
                port: 8080
              };
              this.networkPolicyMaps.set(link.id, this.viewNetwork);
            }
            setTimeout(() => {
              jQuery('#link-path-' + link.id).addClass('app');
            });
          }
        });

        // routes
        // this.routes = [{linkId: '', service: '', path: ''}];
        let idx = 0;
        for(var route of composeRoutes) {
          this.routes.push({
            linkId: idx,
            service: route.serviceName,
            path: route.path
          });
          idx++;
        }
        if(this.routes.length == 0){
          this.routes = [{linkId: '', service: '', path: ''}];
        }

        // configs
        for(let property of composeProperties) {
          this.configs.push({app: property.appName, property: property.properties});
        }
        if(this.configs.length == 0) {
          this.configs.push({app: '', property: ''});
        }

        setTimeout(() => {
          this.directive.nodeSimulation(_x, _y, this.nodes, this.links);
          this.appDropDirective.svgMouseDown();
        });
      }
    );

  }

  zoom(direction) {
    this.directive.zoomClick(direction);
  }
  openMenu(event) {
    this.accordion = true;
    jQuery('.fifteen').prop('class', 'ten wide column');

    var item = event.target;
    if(item.classList.contains("icon") == true) item = event.target.parentElement;
    var activeTab = item.dataset.tab;
    switch(activeTab) {
      case 'app': {
        this.listMarketApps(this.searchAppName);
        break;
      }
      case 'service': {
        this.listMarketServices(this.searchServiceName);
        break;
      }
      case 'network': {
        this.initViewNetwork();
        break;
      }
      case 'routing': {
        break;
      }
      case 'config': {
        break;
      }
      case 'api': {
        this.listApis('');
        break;
      }
    }
  }
  closeMenu() {
    this.accordion = false;
    jQuery('.ten').prop('class', 'fifteen wide column');
    jQuery('.menu .item').removeClass('active');
  }

  searchApps(name: string) {
    this.marketApps = [];
    this.listMarketApps(name);
  }
  searchServices(label: string) {
    this.marketServices = [];
    this.listMarketServices(label);
  }
  searchApis(name: string) {
    this.Apis = [];
    this.listApis(name);
  }
  listMarketApps(name: string) {
    if(this.marketApps.length > 0) return;
    this.marketApps = [];
    let route = 'apps/env?env=' + environment.cfEnvNameMSA+'&private=' + environment.cfEnvNamePrivate;
    if(name) {
      route = route + '&name=' + name;
    }
    this.apiService.get(route).subscribe(
      data => {
        var cf_apps = data['resources'];
        if(cf_apps == null) return false;
        for(var cf_app of cf_apps) {
          if(!cf_app.entity.name.startsWith('gatewayapp-micro')) {
            this.marketApps.push({
              id: cf_app.metadata.guid,
              name: cf_app.entity.name,
              state: cf_app.entity.state
            });
          }
        }
      }
    );
  }

  listMarketServices(label: string) {
    if(this.marketServices.length > 0) return;
    this.marketServices = [];
    let route = 'marketplace';
    if(label) {
      route = route + '?q=label:' + label;
    }
    this.apiService.get(route).subscribe(
      data => {
        var cf_services = data['resources'];
        if(cf_services == null) return false;
        for(var cf_service of cf_services) {
          if(cf_service.entity.label == environment.configServiceLabel || cf_service.entity.label == environment.registryServiceLabel) {
            continue;
          }
          this.marketServices.push({
            id: cf_service.metadata.guid,
            label: cf_service.entity.label,
            plans: cf_service.entity.service_plans
          });
        }
      }
    );
  }

  listApis(name: string) {
    if(this.Apis.length > 0) return;
    this.Apis = [];
    let route = 'apps/env?env=' + environment.cfEnvNameMSA;
    if(name) {
      route = route + '&name=' + name;
    }
    this.apiService.get<any>('apigateway').subscribe(
      data => {
        if(data) {
          for (var d of data) {
            this.Apis.push({
              id: d.id,
              name: d.name
            });
          }
        }
      }
    );
  }

  listAppApis() {
    this.apiService.get<MicroApi[]>(`apigateway/${this.micro.id}/api`).subscribe(
      data => {
        this.microapis = data;
      }
    );
  }

  delAppApi(id: number, microid: number) {
    if(!confirm("삭제하시겠습니까?")) {
      return;
    }

    this.apiService.delete<MicroApi>(`apigateway/${id}/api?microid=${microid}`).subscribe(
      data => {
        console.log(data)
        alert("삭제되었습니다.");
        this.listAppApis();
      }
    );
  }

  showAddApi(microApi: MicroApi){
    jQuery('.ui.modal.addapi')
      .modal({
        inverted: true
      })
      .modal('show')
    ;
    this.selectedMicroapi.username = "";
    this.selectedMicroapi.userpassword = "";
    this.selectedMicroapi = microApi;
  }

  getSwagger(microApi: MicroApi) {
    // this.swaggerApiName = microApi.name;
    jQuery('.ui.modal.swagger')
      .modal({
        inverted: true
      })
      .modal('show')
    ;
    if (this.micro.id != 0) {
      this.apiService.get<MicroApi>(`apigateway/${microApi.id}`).subscribe(
        data => {
          this.iframeSrc = this.sanitizer.bypassSecurityTrustResourceUrl(this.swaggerApiUrl + '/entry/?id=' + data.microId + '&domain=' + this.api);
        }
      );
    }
  }

  addApi(){

    if (this.micro.id != 0 && this.selectedMicroapi.username != "" && this.selectedMicroapi.userpassword != "") {
      this.selectedMicroapi.microId = this.micro.id;

      this.apiService.post<MicroApi>(`apigateway/${this.micro.id}/api`, this.selectedMicroapi).subscribe(
        data => {
          this.listAppApis();
        },
        err => {
          alert(err.error);
        }
      );
    }

  }

  putNodes(node) {
    this.nodes.push(node);
  }
  infoNode(node) {
    jQuery('.ui.mini.modal').modal({inverted: true}).modal('show');
    let data = this.nodeDatas.get(node.id);
    this.modal['type'] = node.type;
    if(data && data.entity) {
      this.modal['node_name'] = data.entity.name;
      if (node.type == 'App') {
        this.modal['instances'] = data.entity.instances;
        this.modal['memory'] = data.entity.memory;
        this.modal['disk_quota'] = data.entity.disk_quota;
      } else if (node.type == 'Service') {
        this.modal['service_plan_guid'] = data.entity.service_plan_guid;
      }
    }
  }
  removeNode(node) {
    if(node.name.startsWith('gatewayapp-micro')){
      alert('삭제할 수 없습니다.');
      return;
    }
    if(!confirm('삭제하시겠습니까?')) {
      return;
    }
    for(var i = 0; i < this.nodes.length; i++) {
      if(this.nodes[i].id === node.id) {
        if(!this.nodes[i].id.startsWith('INITIAL_')) {
          this.delNodes.push(this.nodes[i]); // 삭제노드
        }
        this.nodes.splice(i,1);
        for(var j = 0; j < this.links.length; j++) {
          if(this.links[j].sNode['id'] == node.id || this.links[j].tNode['id'] == node.id) {
            this.links.splice(j,1);
            j = j - 1;
          }
        }
        return false;
      }
    }
  }
  putLink(link) {
    let sourceNode;
    let targetNode;
    for(var node of this.nodes) {
      if(node.id == link.sNode.id) {
        sourceNode = node;
      }
      if(node.id == link.tNode.id) {
        targetNode = node;
      }
    }
    if(sourceNode.type == environment.nodeTypeService) {
      alert('Service에서 연결할 수 없습니다.');
      this.links.splice(this.links.indexOf(link), 1);
    }
    /*if(targetNode.name.indexOf('config-server') != -1 || targetNode.name.indexOf('registry-server') != -1) {
     alert(targetNode.name + '은(는) 자동 연결됩니다.');
     this.links.splice(this.links.indexOf(link), 1);
     }*/
    // app -> service
    if(sourceNode.type == environment.nodeTypeApp && targetNode.type == environment.nodeTypeService) {
      link.type = environment.nodeTypeService;
      if(!sourceNode.name.startsWith("gatewayapp")) {
        this.bindings.push({app: sourceNode.id, service: targetNode.id});
      }
    }
    // app <-> gatewayApp : ass network policy
    if(sourceNode.type == environment.nodeTypeApp && targetNode.type == environment.nodeTypeApp) {
      this.viewNetwork = {
        id: link.id,
        source: sourceNode,
        target: targetNode,
        protocol: 'tcp',
        port: 8080
      };
      this.networkPolicyMaps.set(link.id, this.viewNetwork);
      setTimeout(() => {
        document.getElementById('link-path-'+link.id).dispatchEvent(new Event('click'));
      });
      // gateway -> app : add route
      if (sourceNode.name.startsWith("gatewayapp")) {
        let idx = this.routes.length - 1;
        if (this.routes[idx].linkId == '' && this.routes[idx].service == '') {
          this.routes[idx] = {
            linkId: link.id,
            service: targetNode.name,
            path: '/' + targetNode.name + '/**'
          };
          this.addRoute();
        } else {
          this.routes.push({
            linkId: link.id,
            service: targetNode.name,
            path: '/' + targetNode.name + '/**'
          });
        }
      }
    }
  }
  getLink(link) {
    if(link.type != environment.nodeTypeService) {
      this.tabNetwork.nativeElement.click();
      this.viewNetwork = this.networkPolicyMaps.get(link.id);
    }
  }

  initViewNetwork() {
    let node = new Node();
    node.name = '';
    this.viewNetwork = {id: '', source: node, target: node, protocol: "", port: ""};
  }
  deleteNetwork(network) {
    if(!confirm("삭제하시겠습니까?")){
      return;
    }
    if(network.source.id == undefined) {
      return;
    }
    for(let i = 0; i < this.links.length; i++) {
      if(this.links[i].id == this.viewNetwork.id) {
        this.links.splice(i, 1);
      }
    }
    this.networkPolicyMaps.delete(this.viewNetwork.id);
    this.closeMenu();
    this.initViewNetwork();
  }

  addRoute() {
    this.routes.push({linkId: '', service: '', path: ''});
  }
  removeRoute(route) {
    this.routes.splice(this.routes.indexOf(route), 1);
    /*for(let link of this.links) {
     if(route.linkId == link.id) {
     this.links.splice(this.links.indexOf(link), 1);
     break;
     }
     }*/
  }

  addConfig() {
    this.configs.push({app: '', property: ''});
  }
  deleteConfig(index) {
    this.configs.splice(index,1);
  }

  save() {
    if( !confirm("저장하시겠습니까 ?") ) {
      return;
    }
    let services = {resources: []};
    services.resources.push({
      metadata: {guid: this.configService.metadata.guid},
      entity: {name: this.configService.entity.name, service_plan_guid: this.configService.entity.service_plan_guid}
    });
    services.resources.push({
      metadata: {guid: this.registryService.metadata.guid},
      entity: {name: this.registryService.entity.name, service_plan_guid: this.registryService.entity.service_plan_guid}
    });
    let apps = {resources: []};
    let serviceBindings = {resources: []};
    let networkPolicies = [];
    let routes = [];
    let configs = [];
    for(var node of this.nodes) {
      if(node.type == 'Service') {
        let servicePlanGuid = '';
        for(var marketService of this.marketServices) {
          if(node.id == 'INITIAL_'+marketService.id) {
            if(marketService.plans.length > 0) {
              servicePlanGuid = marketService.plans[0];
            }
            break;
          }
        }
        services.resources.push({metadata: {guid: node.id}, entity: {name: node.name, service_plan_guid: servicePlanGuid}});
      } else if(node.type == 'App') {
        apps.resources.push({metadata: {guid: node.id}, entity: {name: node.name}});
        serviceBindings.resources.push({entity :{app_guid: node.id, service_instance_guid: this.configService.metadata.guid}});
        serviceBindings.resources.push({entity :{app_guid: node.id, service_instance_guid: this.registryService.metadata.guid}});
      }
    }
    for(var binding of this.bindings) {
      serviceBindings.resources.push({entity :{app_guid: binding.app, service_instance_guid: binding.service}});
    }
    // for(let app of apps.resources){
      // for(let binding of this.bindings){
      //   if(app.metadata.guid == binding.app){
      //     app.entity.state = 'back';
      //   }
      // }
    // }
    for(let key of Array.from( this.networkPolicyMaps.keys() )) {
      let policy = {
        source: {
          id: this.networkPolicyMaps.get(key).source.id
        },
        destination: {
          id: this.networkPolicyMaps.get(key).target.id,
          ports: {
            start: this.networkPolicyMaps.get(key).port,
            end: this.networkPolicyMaps.get(key).port
          },
          protocol: this.networkPolicyMaps.get(key).protocol
        }
      };
      networkPolicies.push(policy);

      // frontend
      for(let node of this.nodes){
        if(node.id == policy.destination.id && node.name.startsWith("gatewayapp-micro")) {
          for(let app of apps.resources){
            if(app.metadata.guid == policy.source.id){
              app.entity.state = 'front';
            }
          }
        }
      }
    }
    for(let route of this.routes) {
      routes.push({service: route.service, path: route.path});
    }
    let configMap: Map<string, any> = new Map<string, any>();
    for(let config of this.configs) {
      let properties = [];
      if(configMap.get(config['app']) != undefined) {
        properties = configMap.get(config['app']);
      }
      let property = {};
      property[config['property'].split('=')[0]] = config['property'].split('=')[1];
      properties.push(property);
      configMap.set(config['app'], properties);
    }
    configMap.forEach((value: JSON, key: string) => {
      configs.push({app:key, property:value});
    });
    var status = this.micro['status'];
    if(status == 'INITIAL') {
      status = 'STOPPED';
    }

    // 삭제노드
    let delapps = {resources: []};
    let delservices = {resources: []};
    for(var node of this.delNodes) {
      if(node.type == 'Service') {
        delservices.resources.push({metadata: {guid: node.id}, entity: {name: node.name, service_plan_guid: ''}});
      } else if(node.type == 'App') {
        delapps.resources.push({metadata: {guid: node.id}, entity: {name: node.name}});
      }
    }

    let compose = {
      id: this.micro.id,
      name: this.micro.name,
      orgGuid: this.micro['orgGuid'],
      spaceGuid: this.micro['spaceGuid'],
      status: status,
      version: this.micro.version,
      visible: this.micro.visible,
      composition: {
        services: services,
        apps: apps,
        serviceBindings: serviceBindings,
        policies: networkPolicies,
        routes: routes,
        configs: configs,
        delapps: delapps,
        delservices: delservices,
      }
    };

    this.apiService.put('microservices/'+this.micro.id+'/composition', compose).subscribe(
      res => {
        alert("저장되었습니다.");
        if(this.micro.status == 'INITIAL') {
          this.micro.status = 'STOPPED';
        }
        this.delNodes = [];
      }, err => {
        console.log(JSON.stringify(err.headers));
        alert(err.status+" "+err.message);
      }
    );

  }

  start() {
    let data = {
      name: this.micro['name'],
      spaceGuid: this.micro['spaceGuid'],
      status: 'STARTED'
    };
    this.apiService.put('microservices/'+this.micro.id+'/state', data).subscribe(
      res => {
        if(confirm("시작되었습니다. 상세조회 화면으로 이동하시겠습니까?")) {
          this.router.navigate(['/detail/'+this.micro.id]);
        }
        this.micro.status = 'STARTED';
      }, err => {
        console.log(JSON.stringify(err.headers));
        alert(err.status+" "+err.message);
      }
    );
  }

  delete() {
    if( confirm("마이크로서비스를 삭제하시겠습니까 ?") ) {
      this.apiService.delete('microservices/' + this.micro.id).subscribe(
        res => {
          alert("삭제되었습니다.");
          this.router.navigate(['list']);
        }, err => {
          console.log(JSON.stringify(err.headers));
          console.log(err.status + " " + err.message);
        }
      );
    }
  }

}
