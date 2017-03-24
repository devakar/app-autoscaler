'use strict'
var logger = require('../log/logger');
var _ = require('underscore');
var planValidator = require('./planValidator');
module.exports = function(req, res, next) {  
  planValidator.validatePlan(req.body, req.params.plan_id, function(palnValidationResult) {
    if(!_.isEmpty(planValidationResult)) {
      logger.error('Input policy JSON schema structure is not valid',
      { 'app id': req.params.app_id, 'plan id' : req.params.plan_id, 'error':planValidationResult });
      next (planValidationResult);
    }
    else{
      logger.info('Input policy JSON schema structure have valid plan. Creating policy..',{ 'app id': req.params.app_id });
      next();
    }
  });
};
