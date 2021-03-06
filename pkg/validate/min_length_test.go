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

package validate

import (
	"testing"

	"github.com/smartystreets/assertions"
	"go.thethings.network/lorawan-stack/v3/pkg/util/test/assertions/should"
)

func TestMinLength(t *testing.T) {
	a := assertions.New(t)

	minLength2 := MinLength(2)

	a.So(minLength2(""), should.NotBeNil)
	a.So(minLength2("a"), should.NotBeNil)
	a.So(minLength2("aa"), should.BeNil)

	var slice []int
	a.So(minLength2(slice), should.NotBeNil)
	slice = append(slice, 1)
	a.So(minLength2(slice), should.NotBeNil)
	slice = append(slice, 2)
	a.So(minLength2(slice), should.BeNil)

	a.So(minLength2(nil), should.NotBeNil)
	a.So(minLength2(1), should.NotBeNil)
}
