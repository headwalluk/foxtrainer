package cityhash

import "testing"

// Expected values come from a Python port cross-checked byte-for-byte against Mozilla's compiled city.cpp.
func TestHash64EveryLengthBranch(test *testing.T) {
	cases := []struct {
		length int
		want   uint64
	}{
		{length: 0, want: 0x9ae16a3b2f90404f},
		{length: 1, want: 0x5068a5b3d87a0284},
		{length: 2, want: 0x6ea90655f7a8ea3c},
		{length: 3, want: 0xf6aa543ca4b8bf14},
		{length: 4, want: 0x9dd33b80b9fa6393},
		{length: 7, want: 0x0e1bc1022a1b992c},
		{length: 8, want: 0x37fed2dba3a300e3},
		{length: 9, want: 0x29b1ff616f22d6f6},
		{length: 15, want: 0xf35205fd9db2c504},
		{length: 16, want: 0xbdf5bcbc9b4603a4},
		{length: 17, want: 0xa75fe55f0d775eda},
		{length: 24, want: 0x7353f72904323f52},
		{length: 31, want: 0xa74e1d551760088d},
		{length: 32, want: 0xd7989f47d40c660d},
		{length: 33, want: 0xef04fa08e8e5b8fb},
		{length: 48, want: 0xa7b0eb0c7882e506},
		{length: 63, want: 0x4cca3400afafeff4},
		{length: 64, want: 0xc1515868c51fc399},
		{length: 65, want: 0xf16772a32b6c31d8},
		{length: 100, want: 0x6641e32ed7481060},
		{length: 127, want: 0x962f7a27804c48f1},
		{length: 128, want: 0xcfbb39443ec19617},
		{length: 129, want: 0x94c7aa9883f91f7f},
		{length: 200, want: 0x1320c268cd285bfe},
		{length: 257, want: 0x79f303a6c159065a},
	}

	for _, testCase := range cases {
		data := make([]byte, testCase.length)
		for index := range data {
			data[index] = byte(index*7 + 3)
		}

		if got := Hash64(data); got != testCase.want {
			test.Errorf("length %d: got %#016x, want %#016x", testCase.length, got, testCase.want)
		}
	}
}
