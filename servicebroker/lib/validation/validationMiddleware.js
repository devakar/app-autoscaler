'use strict'
var logger = require('../log/logger');
var _ = require('underscore');
var planValidator = require('./planValidator');

module.exports = function(req, res, next) {  
  planValidator.validatePlan(req, function(planValidationResult) {
    if(!_.isEmpty(planValidationResult)) {
      logger.error('Input policy JSON schema structure is not specific as per service plan',
      { 'app id': req.body.app_guid, 'plan_id': req.body.plan_id, error':planValidationResult });
      next (planValidationResult);
    }
    else{
      logger.info('Input policy JSON is valid as per plan. Checking for JSON schema structure ....',
      { 'app id': req.body.app_guid, 'plan_id': req.body.plan_id });
      next();
    }
  });
};
