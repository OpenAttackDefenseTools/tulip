#!/usr/bin/env python3
# -*- coding: utf-8 -*-

# This file is part of Flower.
#
# Copyright ©2018 Nicolò Mazzucato
# Copyright ©2018 Antonio Groza
# Copyright ©2018 Brunello Simone
# Copyright ©2018 Alessio Marotta
# DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS FILE HEADER.
# 
# Flower is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
# 
# Flower is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with Flower.  If not, see <https://www.gnu.org/licenses/>.

from flask import Flask, Response

from configurations import services
from data2req import convert_http_requests
from base64 import b64decode
from db import DB
from bson import json_util
from flask_cors import CORS
from flask import request

from flow2pwn import flow2pwn

application = Flask(__name__)
CORS(application)
db = DB()


def return_json_response(object):
    return Response(json_util.dumps(object), mimetype='application/json')

def return_text_response(object):
    return Response(object, mimetype='text/plain')


@application.route('/')
def hello_world():
    return 'Hello, World!'


@application.route('/query', methods=['POST'])
def query():
    json = request.get_json()
    result = db.getFlowList(json)
    return return_json_response(result)


@application.route('/starred', methods=['POST'])
def getStarred():
    json = request.get_json()
    json["starred"] = 1
    result = db.getFlowList(json)
    return return_json_response(result)




@application.route('/star/<flow_id>/<star_to_set>')
def setStar(flow_id, star_to_set):
    db.setStar(flow_id, star_to_set)
    return "ok!"


@application.route('/services')
def getServices():
    return return_json_response(services)


@application.route('/flow/<id>')
def getFlowDetail(id):
    to_ret = return_json_response(db.getFlowDetail(id))
    return to_ret


@application.route('/to_python_request', methods=['POST'])
def convertToRequests():
    data = b64decode(request.data)
    tokenize = request.args.get("tokenize", False)
    use_requests_session = request.args.get("use_requests_session", False)
    try:
        converted = convert_http_requests(data, tokenize, use_requests_session)
    except Exception as ex:
        return return_text_response("There was an error while converting the request:\n{}: {}".format(type(ex).__name__, ex))
    return return_text_response(converted)

@application.route('/to_pwn/<id>')
def confertToPwn(id):
    flow = db.getFlowDetail(id)
    converted = flow2pwn(flow)
    return return_text_response(converted)

if __name__ == "__main__":
    application.run(host='0.0.0.0',threaded=True)

