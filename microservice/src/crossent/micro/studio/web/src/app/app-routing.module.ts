import { NgModule } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';
import { MicroListComponent } from './view/micro-list/micro-list.component'
import { MicroDetailComponent } from './view/micro-detail/micro-detail.component'
import { CreateComponent } from './compose/create/create.component';
import { EditComponent } from './compose/edit/edit.component';
import { LoginComponent } from './login/login.component';
import { MicroGuideComponent } from './view/micro-guide/micro-guide.component';
import { ApiListComponent } from './api/api-list/api-list.component';
import { ApiCreateComponent } from './api/api-create/api-create.component';
import { ApiViewComponent } from './api/api-view/api-view.component';
import { ApiHealthComponent } from './api/api-health/api-health.component';
import { ApiManageComponent } from './api/api-manage/api-manage.component';
import { AuthGuard } from './shared/auth.guard';
import { RoleGuard } from './shared/role.guard';
import {ApiUserListComponent} from "./api/api-user-list/api-user-list.component";

const routes: Routes = [
  { path: 'list', component: MicroListComponent, canActivate: [AuthGuard] },
  { path: 'guide', component: MicroGuideComponent, canActivate: [AuthGuard] },
  { path: 'apilist', component: ApiListComponent, canActivate: [AuthGuard] },
  { path: 'apicreate', component: ApiCreateComponent, canActivate: [RoleGuard], data: {expectedRole: 'MANAGER'}},
  { path: 'apiview/:id', component: ApiViewComponent, canActivate: [AuthGuard] },
  { path: 'apiuserlist/:id', component: ApiUserListComponent, canActivate: [AuthGuard] },
  { path: 'apiedit/:id', component: ApiCreateComponent, canActivate: [RoleGuard], data: {expectedRole: 'MANAGER'} },
  { path: 'apihealth', component: ApiHealthComponent, canActivate: [RoleGuard], data: {expectedRole: 'MANAGER'} },
  { path: 'apimanage', component: ApiManageComponent, canActivate: [RoleGuard], data: {expectedRole: 'MANAGER'} },
  { path: 'detail/:id', component: MicroDetailComponent, canActivate: [AuthGuard] },
  { path: 'create', component: CreateComponent, canActivate: [AuthGuard] },
  { path: 'edit/:id', component: EditComponent, canActivate: [AuthGuard] },
  { path: 'login', component: LoginComponent },
  { path: '', redirectTo: 'list', pathMatch: 'full' }
];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule]
})
export class AppRoutingModule { }
