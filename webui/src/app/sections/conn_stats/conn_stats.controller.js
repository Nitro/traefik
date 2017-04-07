'use strict';
var d3 = require('d3');

/** @ngInject */
function ConnStatsController($scope, $interval, $log, $filter, ConnStats) {
  var vm = this,
    refreshInterval = 2000, // milliseconds
    // Store and display data for the past hour
    maxHistoricDataPointCount = 3600 / (refreshInterval / 1000),
    dataPointStorageTime = 3600 * 1000;

  vm.graph = {
    historicalConnCount: {},
    currentConnCount: {}
  };

  vm.graph.historicalConnCount.options = {
    chart: {
      type: 'lineChart',
      height: 300,
      margin: {
        top: 20,
        right: 40,
        bottom: 40,
        left: 55
      },
      xScale: d3.time.scale(),
      xAxis: {
        tickFormat: function (d) {
          return d3.time.format('%X')(new Date(d));
        }
      },
      forceY: [0, 1], // This prevents the chart from showing -1 on Oy when all the input data points
                      // have y = 0. It won't disable the automatic adjustment of the max value.
      useInteractiveGuideline: true,
      duration: 0 // Bug: Markers will not be drawn if you set this to some other value...
    },
    title: {
      enable: true,
      text: 'Historical Connection Counts'
    }
  };

  vm.graph.currentConnCount.options = {
    chart: {
      type: 'discreteBarChart',
      height: 300,
      margin: {
        top: 20,
        right: 40,
        bottom: 40,
        left: 55
      },
      x: function(d) { return d.label; },
      y: function(d) { return d.value; },
      valueFormat: d3.format('d'),
      showValues: true,
      staggerLabels: true
    },
    title: {
      enable: true,
      text: 'Current Connection Counts'
    }
  };

  function updateHistoricalConnCountGraph() {
    var currentDate = Date.now();

    angular.forEach(vm.connStats, function(provider, providerId) {
      angular.forEach(provider.backends, function(backend, backendId) {
        var dataPointKey = providerId + '/' + backendId,
            newDataPoint = {
              x: currentDate,
              y: backend['total_conn']
            };

        // Check if the new data point belongs to an existing plot
        var existingPlot = $filter('filter')(vm.graph.historicalConnCount.data, {'key': dataPointKey})
        if (existingPlot.length > 0) {
          // There should be only one
          existingPlot[0].values.push(newDataPoint);
          existingPlot[0].lastUpdated = currentDate;
        } else {
          vm.graph.historicalConnCount.data.push({
            values: [newDataPoint],
            key: dataPointKey,
            type: 'line',
            lastUpdated: currentDate
          });
        }
      });
    });

    // Limit plot data points
    for (var i = 0; i < vm.graph.historicalConnCount.data.length; i++) {
      var values = vm.graph.historicalConnCount.data[i].values;
      if (values.length > maxHistoricDataPointCount) {
        values.shift();
      }

      // Remove dead entries
      if (values.length == 0 ||
          values[values.length - 1].lastUpdated < Date.now() - dataPointStorageTime) {
        
        vm.graph.historicalConnCount.data.splice(i, 1);
      }
    }
  }

  function updateCurrentConnCountGraph() {
    vm.graph.currentConnCount.data = [{
      values: []
    }];
    
    angular.forEach(vm.connStats, function(provider, providerId) {
      angular.forEach(provider.backends, function(backend, backendId) {
        var dataPointLabel = providerId + '/' + backendId,
            maxConnDataPoint = {
              label: dataPointLabel,
              value: backend['max_conn']
            },
            totalConnDataPoint = {
              label: dataPointLabel,
              value: backend['total_conn']
            };

        vm.graph.currentConnCount.data[0].values.push(maxConnDataPoint);
        vm.graph.currentConnCount.data[0].values.push(totalConnDataPoint);
      });
    });
  }

  function loadData(connStats) {
    // Set the current data point
    vm.connStats = connStats;

    updateHistoricalConnCountGraph();
    updateCurrentConnCountGraph();
  }

  function errData(error) {
    vm.connStats = {};
    $log.error(error);
  }

  // Get the initial data points
  vm.connStats = ConnStats.get(loadData, errData);

  // Initialize the view data for the historical connection count chart
  vm.graph.historicalConnCount.data = [];

  var intervalId = $interval(function () {
      ConnStats.get(loadData, errData);
    },
    refreshInterval
  );

  $scope.$on('$destroy', function () {
    $interval.cancel(intervalId);
  });
}

module.exports = ConnStatsController;
