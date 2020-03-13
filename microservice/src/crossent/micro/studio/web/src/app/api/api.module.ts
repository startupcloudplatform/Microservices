import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { ApiListComponent } from './api-list/api-list.component';
import { ApiCreateComponent } from './api-create/api-create.component';
import { ApiViewComponent } from './api-view/api-view.component';
import { ApiHealthComponent } from './api-health/api-health.component';
import { PartService } from './api-create/part.service';
import { ApiManageComponent } from './api-manage/api-manage.component';
import { CodeFilterPipe } from './api-list/code-filter.pipe';
import { SearchApiFilterPipe } from './api-list/search-filter.pipe';
import { ApiavatarComponent } from './apiavatar/apiavatar.component';
import { BarChartComponent } from './api-health/bar-chart/bar-chart.component';
import { LineChartComponent } from './api-health/line-chart/line-chart.component';
import {ApiUserListComponent} from "./api-user-list/api-user-list.component";

@NgModule({
  imports: [
    FormsModule, CommonModule, RouterModule
  ],
  declarations: [ApiUserListComponent, ApiListComponent, ApiCreateComponent, ApiViewComponent, ApiHealthComponent, ApiManageComponent, CodeFilterPipe, SearchApiFilterPipe, ApiavatarComponent, BarChartComponent, LineChartComponent],
  providers: [ PartService ],
  exports: [ApiavatarComponent, CodeFilterPipe, SearchApiFilterPipe]
})
export class ApiModule { }
