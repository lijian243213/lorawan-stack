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

import { createLogic } from 'redux-logic'

import * as user from '../actions/user'
import api from '../api'
import * as accessToken from '../lib/access-token'

export default [
  createLogic({
    type: user.GET_USER_ME,
    async process ({ getState, action }, dispatch, done) {
      try {
        const result = window.APP_CONFIG.console
          ? await api.v3.is.users.me()
          : await api.oauth.me()
        dispatch(user.getUserMeSuccess(result.data))
      } catch (error) {
        dispatch(user.getUserMeFailure())
      }
      done()
    },
  }),
  createLogic({
    type: user.LOGOUT,
    async process ({ getState, action }, dispatch, done) {
      try {
        if (window.APP_CONFIG.console) {
          await api.console.auth.logout()
        } else {
          await api.oauth.logout()
        }
        accessToken.clear()
        dispatch(user.logoutSuccess())
      } catch (error) {
        dispatch(user.logoutFailure())
      }

      done()
    },
  }),
]
