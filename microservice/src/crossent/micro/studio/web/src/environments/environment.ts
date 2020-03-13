// The file contents for the current environment will overwrite these during build.
// The build system defaults to the dev environment which uses `environment.ts`, but if you do
// `ng build --env=prod` then `environment.prod.ts` will be used instead.
// The list of which env maps to which file can be found in `.angular-cli.json`.
//
export const environment = {
  production: false,
  apiUrl: 'http://localhost:8080/api/v1',
  swaggerApiUrl: 'http://localhost:8080/swagger/',
  cfEnvNameMSA: 'msa',
  cfEnvNamePrivate: 'private',
  msServices: 'config-server,registry-server,gateway-server',
  sampleApps: 'front,back',
  nodeTypeApp: 'App',
  nodeTypeService: 'Service',
  configService: 'config-server',
  registryService: 'registry-server',
  configServiceLabel: 'micro-config-server',
  registryServiceLabel: 'micro-registry-server'
};
