// Copyright © 2019 The Things Network Foundation, The Things Industries B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import {
  GET_DEVICES_LIST,
  SEARCH_DEVICES_LIST,
  GET_DEVICES_LIST_SUCCESS,
  GET_DEVICES_LIST_FAILURE,
} from '../actions/devices'

const defaultState = {
  fetching: true,
  fetchingSearch: false,
  error: undefined,
  devices: [],
  totalCount: 0,
}

const devices = function (state = defaultState, action) {
  switch (action.type) {
  case GET_DEVICES_LIST:
    return {
      ...state,
      fetching: true,
    }
  case SEARCH_DEVICES_LIST:
    return {
      ...state,
      fetching: true,
      fetchingSearch: true,
    }
  case GET_DEVICES_LIST_SUCCESS:
    return {
      ...state,
      fetching: false,
      fetchingSearch: false,
      devices: action.devices,
      totalCount: action.totalCount,
    }
  case GET_DEVICES_LIST_FAILURE:
    return {
      ...state,
      fetching: false,
      fetchingSearch: false,
      error: action.error,
      devices: [],
      totalCount: 0,
    }
  default:
    return state
  }
}

export default devices
