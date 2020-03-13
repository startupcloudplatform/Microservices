import {KeyValue} from "./KeyValue.model";

export class MicroApi {
  constructor(
    public id?: number,
    public part?: string,
    public name?: string,
    public host?: string,
    public path?: string,
    public version?: string,
    public restapi?: string,
    public active?: string,
    public description?: string,
    public image?: string,
    public updated?: string,
    public method?: string,
    public pathStrip?: string,
    public whitelist?: string,
    public headers?: KeyValue[],
    public period?: string,
    public average?: string,
    public burst?: string,
    public maxconn?: string,
    public microId?: number,
    public favorite?: number,
    public username?: string,
    public userpassword?: string,
    public orgguid?: string,
    public microname?: string,
  ) {}
}
