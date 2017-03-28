'use strict'
var logger = require('../logger/logger');
var _ = require('underscore');
var planValidator = require('./planValidator');

module.exports = function(req, res, next) {  
  var service_id = req.body.service_id;
  var plan_id = req.body.plan_id;
  var policy = req.body.parameters;
  planValidator.validatePlan(policy, service_id, plan_id, function(planValidationResult) {
    if(!_.isEmpty(planValidationResult)) {
      logger.error('Input policy JSON schema structure is not specific as per service plan of autoscaler service',
      { 'app id': req.body.app_guid, 'service_id': service_id, 'plan_id': plan_id, 'error': planValidationResult });
      next (planValidationResult);
    }
    else{
      logger.info('Input policy JSON is valid as per plan. Checking for JSON schema structure ....',
      { 'app id': req.body.app_guid, 'service_id': service_id, 'plan_id': plan_id });
      next();
    }
  });
};
