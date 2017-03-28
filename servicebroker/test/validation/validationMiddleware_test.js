'use strict';

var supertest = require("supertest");
var expect = require('chai').expect;
var nock = require('nock');
var uuid = require('uuid');

var fs = require('fs');
var path = require('path');
var BrokerServer = require(path.join(__dirname, '../../lib/server.js'));
var configFilePath = path.join(__dirname, '../../config/settings.json');
var validationMiddleware = require(path.join(__dirname, '../../lib/validation/validationMiddleware'));
var logger = require(path.join(__dirname,'../../lib/logger/logger'));

var settings = require(path.join(__dirname, '../../lib/config/setting.js'))((JSON.parse(
  fs.readFileSync(configFilePath, 'utf8'))));

var models = require(path.join(__dirname,'../../lib/models'))(settings.db);
var serviceInstance = models.service_instance;
var binding = models.binding;

var auth = new Buffer(settings.username + ":" + settings.password).toString('base64');
var scope;

function initNockBind(statusCode) {
  scope = nock(settings.apiserver.uri)
    .put(/\/v1\/policies\/.*/)
    .reply(statusCode, {
      'success': true,
      'error': null,
      'result': "created"
    });
}

describe('Validation of Policy JSON as per specific plan of service', function() {
  var server, serviceInstanceId,  orgId, spaceId, appId, bindingId, policyContent, fakePolicy;

  serviceInstanceId = uuid.v4();
  orgId = uuid.v4();
  spaceId = uuid.v4();
  appId = uuid.v4();
  bindingId = uuid.v4();
  
  var service_condition = {
    'serviceInstanceId': serviceInstanceId,
    'orgId': orgId,
    'spaceId': spaceId,
    where: { 'serviceInstanceId': serviceInstanceId, 'orgId': orgId, 'spaceId': spaceId },
  };
  
  var policy = { "policy": "testPolicy" };

  before(function(done) {
    server = BrokerServer(configFilePath);
    done();
  });

  after(function(done) {
    server.close(done)
  });

  beforeEach(function(done) {
    policyContent = fs.readFileSync(__dirname+'/../fakePolicy.json', 'utf8');
    fakePolicy = JSON.parse(policyContent);
    binding.truncate({ cascade: true }).then(function(result) {
      serviceInstance.truncate({ cascade: true }).then(function(result) {
        serviceInstance.create(service_condition).then(function(result) {
          done();
        });
      });
    });
  });


  it("should validate plan", function(done) {
    initNockBind(201);
    supertest(server)
      .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId, validationMiddleware)
      .set("Authorization", "Basic " + auth)
      .send({ "app_guid": appId, "service_id": "autoscaler-guid", "plan_id": "autoscaler-free-plan-id", "parameters": policy })
      .expect(201)
      .expect('Content-Type', /json/)
      .expect({})
      .end(function(err, res) {
        expect(err).to.be.null;
        expect(res.status).to.equal(201);
        done();
       });
  });
  
  it("should not validate plan as recurring schedules exceeded maximum limit as per plan", function(done) {
    delete fakePolicy.schedules.specific_date;
    delete fakePolicy.scaling_rules;
    supertest(server)
      .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId, validationMiddleware)
      .set("Authorization", "Basic " + auth)
      .send({ "app_guid": appId, "service_id": "autoscaler-guid", "plan_id": "autoscaler-free-plan-id", "parameters": fakePolicy })
      .end(function(err, res) {
        expect({});
        expect(res.status).to.equal(500);
        done();
       });
  });

  it("should not validate plan as specific date schedules exceeded maximum limit as per plan", function(done) {
    delete fakePolicy.schedules.recurring_schedule;
    delete fakePolicy.scaling_rules;
    supertest(server)
      .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId, validationMiddleware)
      .set("Authorization", "Basic " + auth)
      .send({ "app_guid": appId, "service_id": "autoscaler-guid", "plan_id": "autoscaler-free-plan-id", "parameters": fakePolicy })
      .end(function(err, res) {
        expect({});
        expect(res.status).to.equal(500);
        done();
       });
  });

  it("should not validate plan as scaling rules exceeded maximum limit as per plan", function(done) {
    delete fakePolicy.schedules.recurring_schedule;
    delete fakePolicy.scaling_rules;
    supertest(server)
      .put("/v2/service_instances/" + serviceInstanceId + "/service_bindings/" + bindingId, validationMiddleware)
      .set("Authorization", "Basic " + auth)
      .send({ "app_guid": appId, "service_id": "autoscaler-guid", "plan_id": "autoscaler-free-plan-id", "parameters": fakePolicy })
      .end(function(err, res) {
        expect({});
        expect(res.status).to.equal(500);
        done();
       });
  });
});