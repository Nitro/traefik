'use strict';
var angular = require('angular');
var traefikCoreConnStats = require('../../core/conn_stats.resource');
var ConnStatsController = require('./conn_stats.controller');

var traefikConnStats = 'traefik.conn_stats';
module.exports = traefikConnStats;

angular
  .module(traefikConnStats, [traefikCoreConnStats])
  .controller('ConnStatsController', ConnStatsController)
  .config(config);

  /** @ngInject */
  function config($stateProvider) {

    $stateProvider.state('conn_stats', {
      url: '/conn_stats',
      template: require('./conn_stats.html'),
      controller: 'ConnStatsController',
      controllerAs: 'connStatsCtrl'
    });

  }
