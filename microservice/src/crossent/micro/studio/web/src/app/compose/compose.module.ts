import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router'
import { FormsModule } from "@angular/forms";
import { CreateComponent } from './create/create.component';
import { EditComponent } from './edit/edit.component';
import { D3StudioModule } from '../d3-studio/d3-studio.module';

@NgModule({
  imports: [
    CommonModule, RouterModule, FormsModule, D3StudioModule
  ],
  declarations: [CreateComponent, EditComponent]
})
export class ComposeModule { }
