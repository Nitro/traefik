'use strict';
var angular = require('angular');
require('nvd3');
var ndv3 = require('angular-nvd3');
var traefikSectionHealth = require('./health/health.module');
var traefikSectionProviders = require('./providers/providers.module');
var traefikSectionConnStats = require('./conn_stats/conn_stats.module');

var traefikSection = 'traefik.section';
module.exports = traefikSection;

angular
  .module(traefikSection, [
    'ui.router',
    'ui.bootstrap',
    ndv3,
    traefikSectionProviders,
    traefikSectionHealth,
    traefikSectionConnStats
   ])
  .config(config);

  /** @ngInject */
  function config($urlRouterProvider) {
    $urlRouterProvider.otherwise('/');
  }