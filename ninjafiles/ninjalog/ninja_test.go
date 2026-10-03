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

package ninjalog

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const sampleNinjaLog = `# ninja log v5
2629549	2637970	1734113329201554484	out/soong/.intermediates/external/perfetto/libperfetto_c/android_arm64_armv8-2a_cortex-a55_shared_fuzzer_libfuzzer/obj/.intermediates/external/perfetto/perfetto_protos_perfetto_config_android_cpp_gen/gen/external/perfetto/protos/perfetto/config/android/packages_list_config.gen.o	d8b2159292446329
7430437	7430457	1734130381978202868	out/dist/com.android.neuralnetworks-installed-files.txt	b8b53a4e45ad2000
3455625	3459148	1733757682921371065	out/soong/.intermediates/frameworks/av/media/module/codecs/amrnb/dec/libstagefright_amrnbdec/linux_glibc_x86_static/obj/frameworks/av/media/module/codecs/amrnb/dec/src/agc.o	eb99fc1e9eb40ba6
3298321	3298354	1733853322115437442	out/dist/licensetexts/NotoSans-Regular.ttf.txt	d66f829c932e4c29
8040790	8040928	1734118739953088762	out/soong/.intermediates/frameworks/base/api/android.stubs.source/android_common/released-flagged-apis-exportable.txt	4e32fdf9f516d327
6748934	6748942	1734142015835825034	out/soong/.intermediates/cts/tests/framework/base/windowmanager/dndtargetapp/CtsDragAndDropTargetApp/android_common/aapt2/cts/tests/framework/base/windowmanager/dndtargetapp/res/layout_target_activity.xml.flat	fff5530fee39e567
`

func TestParseNinjaLog(t *testing.T) {
	logs, err := parse(strings.NewReader(sampleNinjaLog))

	require.NoError(t, err)
	require.Len(t, logs, 6)
	require.Equal(t, 6748942*time.Millisecond, logs[5].End)
}

func TestParseWrongHeader(t *testing.T) {
	ninjaLog := `# ninja log v6
2629549 2637970 1734113329201554484     out/soong/.intermediates/external/perfetto/libperfetto_c/android_arm64_armv8-2a_cortex-a55_shared_fuzzer_libfuzzer/obj/.intermediates/external/perfetto/perfetto_protos_perfetto_config_android_cpp_gen/gen/external/perfetto/protos/perfetto/config/android/packages_list_config.gen.o    d8b2159292446329
`
	_, err := parse(strings.NewReader(ninjaLog))

	require.Error(t, err)
}

func TestWriteParse(t *testing.T) {
	f, err := os.CreateTemp("", "ninjago-test-*")
	require.NoError(t, err)
	fileName := f.Name()
	require.NoError(t, f.Close())
	defer os.Remove(fileName)

	logs, err := parse(strings.NewReader(sampleNinjaLog))
	require.NoError(t, err)

	require.NoError(t, err)
	defer func() { f.Close() }()

	nl := New(fileName)
	require.NoError(t, nl.OpenForWrite())

	for _, ent := range logs {
		require.NoError(t, nl.WriteEntry(ent))
	}

	written, err := os.ReadFile(fileName)
	require.NoError(t, err)

	require.Equal(t, sampleNinjaLog, string(written))
}
