// Copyright 2025 Google LLC
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

package parse

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/google/ninja-go/ninja"
	"github.com/google/ninja-go/runtime"
)

func TestParseKeyValue(t *testing.T) {
	s := `key = value`
	kv, i, err := parseKeyValue([]byte(s), 0)
	require.NoError(t, err)
	assert.Equal(t, 11, i)
	assert.Equal(t, "key", kv.Key)
	assert.Equal(t, "value", kv.Value)
}

func TestParseKeyValue2(t *testing.T) {
	s := `key = value
	other lines`
	kv, i, err := parseKeyValue([]byte(s), 0)
	require.NoError(t, err)
	assert.Equal(t, 11, i)
	assert.Equal(t, "key", kv.Key)
	assert.Equal(t, "value", kv.Value)
}

func TestParseKeyValue3(t *testing.T) {
	s := `key = line-1 $
line-2
other-lines`
	kv, i, err := parseKeyValue([]byte(s), 0)
	require.NoError(t, err)
	assert.Equal(t, 21, i)
	assert.Equal(t, "key", kv.Key)
	assert.Equal(t, "line-1 $\nline-2", kv.Value)
}

func TestParseTrailingSpace(t *testing.T) {
	s := `key = line followed by space  
next-line`
	kv, i, err := parseKeyValue([]byte(s), 0)
	require.NoError(t, err)
	assert.Equal(t, "key", kv.Key)
	assert.Equal(t, "line followed by space  ", kv.Value)
	assert.Equal(t, "\nnext-line", s[i:])
}

func TestParseKeyError(t *testing.T) {
	s := `= line-1 $
line-2
other-line`
	_, i, err := parseKeyValue([]byte(s), 0)
	require.Error(t, err)
	assert.Equal(t, 17, i)
}

func TestParser1(t *testing.T) {
	s := `
pool highmem_pool
 depth = 23
subninja out/build-cuttlestone_x86_64_phone.ninja
subninja out/build-cuttlestone_x86_64_phone-package

# first var
key1 = value1
   
#second var
key2 = value2

subninja s1
default 

pool pool-name
    depth = 123

default d1   d2
pool $
    pool-name-2
    depth = 456

default  $
    d3 $
	d4  d5 
key3 = value$


subninja $
	s2

	
build $
        out/soong/.intermediates/external/XNNPACK/xnnpack_avx_bench_microkernels/android_vendor.33_x86_silvermont_static/obj/external/XNNPACK/src/qs8-gemm/gen/3x4c2-minmax-fp32-avx-ld128.tidy.dep: g.cc.clangTidyDep $
        external/XNNPACK/src/qs8-gemm/gen/3x4c2-minmax-fp32-avx-ld128.c | ${ccCmd} || $
        out/soong/.intermediates/bionic/libc/libc/android_vendor.33_x86_silvermont_shared/versioner.timestamp
    description = ${m.xnnpack_avx_bench_microkernels_android_vendor.33_x86_silvermont_static.moduleDesc}clang-tidy-dep src/qs8-gemm/gen/3x4c2-minmax-fp32-avx-ld128.c${m.xnnpack_avx_bench_microkernels_android_vendor.33_x86_silvermont_static.moduleDescSuffix}
    cFlags = ${m.xnnpack_avx_bench_microkernels_android_vendor.33_x86_silvermont_static.cFlags1}
`
	_, n, err := parseSegment("file1", 0, []byte(s), nil)
	require.NoError(t, err)
	assert.Len(t, n.KeyValues, 3)
	assert.Len(t, n.Default, 5)
	assert.Len(t, n.Subninjas, 4)
	assert.Len(t, n.Pools, 3)
	assert.Len(t, n.Builds, 1)
}

func TestParserParseTokens(t *testing.T) {
	s := ` abc c $
	d e f
`
	ss, i, err := parseTokens([]byte(s), 0)
	require.NoError(t, err)
	assert.Len(t, ss, 5)
	assert.Equal(t, 15, i)
}

func NewContext() context.Context {
	return context.WithValue(context.Background(), runtime.RunParamsKey, &runtime.RunParameters{
		CliFlags: &runtime.CliFlags{
			ConcurrencyLevel: 10,
		},
	})
}

func TestNinjaParseEmpty(t *testing.T) {
	_, err := ParseNinjaFile(NewContext(), "test.ninja", []byte(""), nil)
	require.NoError(t, err)
}

func TestNinjaParseError1(t *testing.T) {
	_, err := ParseNinjaFile(NewContext(), "test.ninja", []byte("xyz"), nil)
	require.Error(t, err)
	assert.Equal(t, "error in parsing test.ninja:3 cannot find = sign in a key-value", err.Error())
}

func TestNinjaParseError2(t *testing.T) {
	_, err := ParseNinjaFile(NewContext(), "test.ninja", []byte(" key=value"), nil)
	require.Error(t, err)
}

func TestNinjaParseError4(t *testing.T) {
	_, err := ParseNinjaFile(NewContext(), "test.ninja", []byte("build"), nil)
	require.Error(t, err)
}

func TestNinjaParseError5(t *testing.T) {
	_, err := ParseNinjaFile(NewContext(), "test.ninja", []byte("pool"), nil)
	require.Error(t, err)
}

func TestNinjaParseError6(t *testing.T) {
	_, err := ParseNinjaFile(NewContext(), "test.ninja", []byte("rule"), nil)
	require.Error(t, err)
}

func TestNinjaParseError7(t *testing.T) {
	_, err := ParseNinjaFile(NewContext(), "test.ninja", []byte("subninja"), nil)
	require.Error(t, err)
}

func TestNinjaParseOpenBracket(t *testing.T) {
	_, err := ParseNinjaFile(NewContext(), "test.ninja", []byte("build x-${test} : my-rule"), nil)
	require.NoError(t, err)

	_, err = ParseNinjaFile(NewContext(), "test.ninja", []byte("build x-${test : my-rule"), nil)
	require.Error(t, err)

	_, err = ParseNinjaFile(NewContext(), "test.ninja", []byte("key = value-${var}-end"), nil)
	require.NoError(t, err)

	_, err = ParseNinjaFile(NewContext(), "test.ninja", []byte("key = value-${var-end"), nil)
	require.Error(t, err)
}

func TestNinjaParse1(t *testing.T) {
	n, err := ParseNinjaFile(NewContext(), "test.ninja", []byte(`
ninja_required_version = 1.1
builddir = out
cflags = -DTEST
msvc_deps_prefix = prefix
default d1 d2
pool local_pool
 depth = 74
rule r1
rule r2
  command=echo hello
build :r1
key = value
default d3
subninja out/b1
subninja out/b2
	`), nil)
	require.NoError(t, err)
	n.Sort()
	assert.Equal(t, "out", n.BuildDir)
	assert.Equal(t, "-DTEST", n.CFlags)
	assert.Equal(t, "prefix", n.MSVCDepsPrefix)
	assert.Equal(t, []*ninja.KeyValue{{Key: "key", Value: "value"}}, n.KeyValues)
	assert.Equal(t, []string{"out/b1", "out/b2"}, n.Subninjas)
	assert.Equal(t, []string{"d1", "d2", "d3"}, n.Default)
	assert.Equal(t, []*ninja.Pool{{Name: "local_pool", Depth: 74}}, n.Pools)
	assert.Equal(t, []*ninja.Rule{{Name: "r1"}, {Name: "r2", Command: "echo hello"}}, n.Rules)
}

func TestPoolParseError1(t *testing.T) {
	_, _, err := parsePool([]byte("invalid name"), 0)
	require.Error(t, err)
}

func TestPoolParseError3(t *testing.T) {
	_, _, err := parsePool([]byte(`name
	xxx`), 0)
	require.Error(t, err)
}

func TestPoolParseError4(t *testing.T) {
	_, _, err := parsePool([]byte(`name",
  key=value`), 0)
	require.Error(t, err)
}

func TestPoolParseError5(t *testing.T) {
	_, _, err := parsePool([]byte(`name
  depth=abc`), 0)
	require.Error(t, err)
}

func TestPoolParseSimple(t *testing.T) {
	r, _, err := parsePool([]byte("pool-name"), 0)
	require.NoError(t, err)
	assert.Equal(t, "pool-name", r.Name)
}

func TestPoolParse1(t *testing.T) {
	r, _, err := parsePool([]byte(`pool-name
  depth = 123`), 0)
	require.NoError(t, err)
	assert.Equal(t, "pool-name", r.Name)
	assert.Equal(t, 123, r.Depth)
}

func TestBuildParseError3(t *testing.T) {
	_, _, err := parseBuild([]byte(`name: rule
  xxx`), 0)
	require.Error(t, err)
}

func TestBuildParse1(t *testing.T) {
	b, _, err := parseBuild([]byte("out: rule-name in1 in2"), 0)
	require.NoError(t, err)
	assert.Equal(t, "rule-name", b.RuleName)
	assert.Equal(t, []string{"out"}, b.Outputs)
	assert.Equal(t, []string{"in1", "in2"}, b.Inputs)
}

func TestBuildParse2(t *testing.T) {
	b, _, err := parseBuild([]byte("out1 out2 | imp-out1 imp-out2: rule-name in1 in2 | imp-in1 imp-in2 || order1 order2"), 0)
	require.NoError(t, err)
	assert.Equal(t, "rule-name", b.RuleName)
	assert.Equal(t, []string{"out1", "out2"}, b.Outputs)
	assert.Equal(t, []string{"imp-out1", "imp-out2"}, b.ImplicitOutputs)
	assert.Equal(t, []string{"in1", "in2"}, b.Inputs)
	assert.Equal(t, []string{"imp-in1", "imp-in2"}, b.ImplicitInputs)
	assert.Empty(t, b.KeyValues)
	assert.Equal(t, []string{"order1", "order2"}, b.OrderOnlyInputs)
}

func TestBuildParseContent(t *testing.T) {
	b, _, err := parseBuild([]byte(`out: rule1 input
  pool = pool1
  phony_output = true
  key1 = value1
  key2  =  value2 value3`), 0)
	require.NoError(t, err)
	assert.Equal(t, "rule1", b.RuleName)
	assert.Equal(t, []string{"out"}, b.Outputs)
	assert.Empty(t, b.ImplicitOutputs)
	assert.Equal(t, []string{"input"}, b.Inputs)
	assert.Equal(t, []*ninja.KeyValue{{Key: "pool", Value: "pool1"}, {Key: "phony_output", Value: "true"}, {Key: "key1", Value: "value1"}, {Key: "key2", Value: "value2 value3"}}, b.KeyValues)
}

func TestRuleParseError1(t *testing.T) {
	_, _, err := parseRule([]byte("rule invalid name"), 0)
	require.Error(t, err)
}

func TestRuleParseError3(t *testing.T) {
	_, _, err := parseRule([]byte(`name
	xxx`), 0)
	require.Error(t, err)
}

func TestRuleParseError4(t *testing.T) {
	_, _, err := parseRule([]byte(`name
	key=value`), 0)
	require.Error(t, err)
}

func TestRuleParseSimple(t *testing.T) {
	r, i, err := parseRule([]byte("rule-name"), 0)
	require.NoError(t, err)
	assert.Equal(t, 9, i)
	assert.Equal(t, "rule-name", r.Name)
}

func TestRuleParseSimple2(t *testing.T) {
	r, i, err := parseRule([]byte("rule$ name"), 0)
	require.NoError(t, err)
	assert.Equal(t, 10, i)
	assert.Equal(t, "rule$ name", r.Name)
}

func TestRuleParse1(t *testing.T) {
	r, i, err := parseRule([]byte(`rule-name
  command = echo hello ninja!`), 0)
	require.NoError(t, err)
	assert.Equal(t, 39, i)
	assert.Equal(t, "rule-name", r.Name)
	assert.Equal(t, "echo hello ninja!", r.Command)
}

func TestRuleParse2(t *testing.T) {
	r, _, err := parseRule([]byte(`rule-name
  command = echo hello ninja!
  description = this is a test rule...
  depfile = my-depfile
  deps = gcc
  msvc_deps_prefix = prefix
  dyndep=
  generator = seq 1 100
  pool = p
  restat = my-restat
	rspfile = my-rspfile
	rspfile_content = my-rspfile_content`), 0)
	require.NoError(t, err)
	assert.Equal(t, "rule-name", r.Name)
	assert.Equal(t, "my-depfile", r.DepFile)
	assert.Equal(t, "echo hello ninja!", r.Command)
	assert.Equal(t, "this is a test rule...", r.Description)
	assert.Equal(t, "gcc", r.Deps)
	assert.Equal(t, "prefix", r.MSVCDepsPrefix)
	assert.Equal(t, "", r.DynDep)
	assert.Equal(t, "seq 1 100", r.Generator)
	assert.Equal(t, "p", r.Pool)
	assert.Equal(t, "my-restat", r.Restat)
	assert.Equal(t, "my-rspfile", r.RSPFile)
	assert.Equal(t, "my-rspfile_content", r.RSPFileContent)
}

func TestNinjaParseRandom1(t *testing.T) {
	txt := `
g.android.soong.cc.config.WindowsCflags = -DUSE_MINGW -DWIN32_LEAN_AND_MEAN -Wno-unused-parameter -D__STDC_FORMAT_MACROS -D__STDC_CONSTANT_MACROS -D__USE_MINGW_ANSI_STDIO=1 -D_WIN32_WINNT=0x0601 -DWINVER=0x0601 -D_FILE_OFFSET_BITS=64 -mno-ms-bitfields --sysroot ${g.android.soong.cc.config.WindowsGccRoot}/${g.android.soong.cc.config.WindowsGccTriple}
m.libbrotli_windows_x86_static.cFlags1 = $ ${g.android.soong.cc.config.WindowsCflags} ${g.android.soong.cc.config.WindowsX86Cflags} ${g.android.soong.cc.config.CommonGlobalCflags} ${g.android.soong.cc.config.HostGlobalCflags} ${g.android.soong.cc.config.ExternalCflags} -B${g.android.soong.cc.config.WindowsGccRoot}/${g.android.soong.cc.config.WindowsGccTriple}/bin -target i686-windows-gnu ${g.android.soong.cc.config.CommonGlobalConlyflags} -Iexternal/brotli/c/include -Iexternal/brotli -Werror -O2 -std=gnu99 ${g.android.soong.cc.config.CommonGlobalIncludes} ${g.android.soong.cc.config.WindowsIncludeFlags} ${g.android.soong.cc.config.NoOverrideGlobalCflags} ${g.android.soong.cc.config.NoOverrideExternalGlobalCflags}


g.android.soong.cc.config.LinuxCflags = -Wa,--noexecstack -fPIC -U_FORTIFY_SOURCE -D_FORTIFY_SOURCE=2 -fstack-protector -D__STDC_FORMAT_MACROS -D__STDC_CONSTANT_MACROS --gcc-toolchain=${g.android.soong.cc.config.LinuxGccRoot} -fstack-protector-strong
m.bcm_object_linux_glibc_x86_64.cFlags2 = $ ${g.android.soong.cc.config.LinuxCflags} ${g.android.soong.cc.config.LinuxX8664Cflags} ${g.android.soong.cc.config.LinuxGlibcCflags} ${g.android.soong.cc.config.CommonGlobalCflags} ${g.android.soong.cc.config.HostGlobalCflags} ${g.android.soong.cc.config.ExternalCflags}  -target x86_64-linux-gnu  -D__ASSEMBLY__ -Iexternal/boringssl/src/include -Iexternal/boringssl/src/crypto -Iexternal/boringssl -Iexternal/libcxx/include -Iexternal/libcxxabi/include  ${g.android.soong.cc.config.CommonGlobalIncludes}

build $
		out/soong/.intermediates/packages/modules/AppSearch/service/service-appsearch/android_common_apex10000/jarjar/service-appsearch.jar: g.java.jarjar $
		out/soong/.intermediates/packages/modules/AppSearch/service/service-appsearch/android_common_apex10000/combined/service-appsearch.jar | ${g.android.soong.java.config.JavaCmd} $
		${g.android.soong.java.config.JarjarCmd} ${rulesFile} packages/modules/AppSearch/service/jarjar-rules.txt
	description = ${m.service-appsearch_android_common_apex10000.moduleDesc}jarjar${m.service-appsearch_android_common_apex10000.moduleDescSuffix}
	rulesFile = packages/modules/AppSearch/service/jarjar-rules.txt

build $
		out/soong/.intermediates/packages/modules/AppSearch/service/service-appsearch/android_common_apex10000/package-check/service-appsearch.jar: g.android.Cp $
		out/soong/.intermediates/packages/modules/AppSearch/service/service-appsearch/android_common_apex10000/jarjar/service-appsearch.jar |@ $
		out/soong/.intermediates/packages/modules/AppSearch/service/service-appsearch/android_common_apex10000/package-check.stamp
	`
	n, err := ParseNinjaFile(NewContext(), "file", []byte(txt), nil)
	require.NoError(t, err)
	n.Sort()
	assert.Len(t, n.Builds, 2)
	assert.Len(t, n.Builds[1].Validations, 1)
	assert.Len(t, n.KeyValues, 4)
}

// TestNinjaParseRemoteRules validates we don't fail parsing on the remote rules that were added
// to guide Siso in remote execution.
func TestNinjaParseRemoteRules(t *testing.T) {
	txt := `
rule g.cc.cc
    command = ${g.cc.relPwd} ${g.android.soong.cc.config.CcWrapper}${ccCmd} -c ${cFlags} -MD -MF ${out}.d -o ${out} ${in}${postCmd}
    depfile = ${out}.d
    deps = gcc
    remote_enabled = true
    remote_platform_ref = default
    remote_timeout = 20m
	`
	n, err := ParseNinjaFile(NewContext(), "file", []byte(txt), nil)
	require.NoError(t, err)
	n.Sort()
	assert.Len(t, n.Rules, 1)
	assert.Equal(t, "g.cc.cc", n.Rules[0].Name)
	assert.Equal(t, "gcc", n.Rules[0].Deps)
}

