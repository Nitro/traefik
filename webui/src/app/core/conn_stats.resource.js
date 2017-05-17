'use strict';
var angular = require('angular');

var traefikCoreConnStats = 'traefik.core.conn_stats';
module.exports = traefikCoreConnStats;

angular
  .module(traefikCoreConnStats, ['ngResource'])
  .factory('ConnStats', ConnStats);

  /** @ngInject */
  function ConnStats($resource) {
    return $resource('../api/conn_stats');
  }
