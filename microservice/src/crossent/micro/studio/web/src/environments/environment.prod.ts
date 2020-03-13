export const environment = {
  production: true,
  apiUrl: 'http://10.244.238.2:9088/api/v1',
  swaggerApiUrl: 'http://10.244.238.2:9088/swagger/',
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
