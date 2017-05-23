// DO NOT EDIT!
// Code generated by ffjson <https://github.com/pquerna/ffjson>
// source: services_state.go
// DO NOT EDIT!

package catalog

import (
	"bytes"
	"fmt"
	"github.com/Nitro/sidecar/service"
	fflib "github.com/pquerna/ffjson/fflib/v1"
)

func (mj *ChangeEvent) MarshalJSON() ([]byte, error) {
	var buf fflib.Buffer
	if mj == nil {
		buf.WriteString("null")
		return buf.Bytes(), nil
	}
	err := mj.MarshalJSONBuf(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
func (mj *ChangeEvent) MarshalJSONBuf(buf fflib.EncodingBuffer) error {
	if mj == nil {
		buf.WriteString("null")
		return nil
	}
	var err error
	var obj []byte
	_ = obj
	_ = err
	buf.WriteString(`{"Service":`)

	{

		err = mj.Service.MarshalJSONBuf(buf)
		if err != nil {
			return err
		}

	}
	buf.WriteString(`,"PreviousStatus":`)
	fflib.FormatBits2(buf, uint64(mj.PreviousStatus), 10, mj.PreviousStatus < 0)
	buf.WriteString(`,"Time":`)

	{

		obj, err = mj.Time.MarshalJSON()
		if err != nil {
			return err
		}
		buf.Write(obj)

	}
	buf.WriteByte('}')
	return nil
}

const (
	ffj_t_ChangeEventbase = iota
	ffj_t_ChangeEventno_such_key

	ffj_t_ChangeEvent_Service

	ffj_t_ChangeEvent_PreviousStatus

	ffj_t_ChangeEvent_Time
)

var ffj_key_ChangeEvent_Service = []byte("Service")

var ffj_key_ChangeEvent_PreviousStatus = []byte("PreviousStatus")

var ffj_key_ChangeEvent_Time = []byte("Time")

func (uj *ChangeEvent) UnmarshalJSON(input []byte) error {
	fs := fflib.NewFFLexer(input)
	return uj.UnmarshalJSONFFLexer(fs, fflib.FFParse_map_start)
}

func (uj *ChangeEvent) UnmarshalJSONFFLexer(fs *fflib.FFLexer, state fflib.FFParseState) error {
	var err error = nil
	currentKey := ffj_t_ChangeEventbase
	_ = currentKey
	tok := fflib.FFTok_init
	wantedTok := fflib.FFTok_init

mainparse:
	for {
		tok = fs.Scan()
		//	println(fmt.Sprintf("debug: tok: %v  state: %v", tok, state))
		if tok == fflib.FFTok_error {
			goto tokerror
		}

		switch state {

		case fflib.FFParse_map_start:
			if tok != fflib.FFTok_left_bracket {
				wantedTok = fflib.FFTok_left_bracket
				goto wrongtokenerror
			}
			state = fflib.FFParse_want_key
			continue

		case fflib.FFParse_after_value:
			if tok == fflib.FFTok_comma {
				state = fflib.FFParse_want_key
			} else if tok == fflib.FFTok_right_bracket {
				goto done
			} else {
				wantedTok = fflib.FFTok_comma
				goto wrongtokenerror
			}

		case fflib.FFParse_want_key:
			// json {} ended. goto exit. woo.
			if tok == fflib.FFTok_right_bracket {
				goto done
			}
			if tok != fflib.FFTok_string {
				wantedTok = fflib.FFTok_string
				goto wrongtokenerror
			}

			kn := fs.Output.Bytes()
			if len(kn) <= 0 {
				// "" case. hrm.
				currentKey = ffj_t_ChangeEventno_such_key
				state = fflib.FFParse_want_colon
				goto mainparse
			} else {
				switch kn[0] {

				case 'P':

					if bytes.Equal(ffj_key_ChangeEvent_PreviousStatus, kn) {
						currentKey = ffj_t_ChangeEvent_PreviousStatus
						state = fflib.FFParse_want_colon
						goto mainparse
					}

				case 'S':

					if bytes.Equal(ffj_key_ChangeEvent_Service, kn) {
						currentKey = ffj_t_ChangeEvent_Service
						state = fflib.FFParse_want_colon
						goto mainparse
					}

				case 'T':

					if bytes.Equal(ffj_key_ChangeEvent_Time, kn) {
						currentKey = ffj_t_ChangeEvent_Time
						state = fflib.FFParse_want_colon
						goto mainparse
					}

				}

				if fflib.SimpleLetterEqualFold(ffj_key_ChangeEvent_Time, kn) {
					currentKey = ffj_t_ChangeEvent_Time
					state = fflib.FFParse_want_colon
					goto mainparse
				}

				if fflib.EqualFoldRight(ffj_key_ChangeEvent_PreviousStatus, kn) {
					currentKey = ffj_t_ChangeEvent_PreviousStatus
					state = fflib.FFParse_want_colon
					goto mainparse
				}

				if fflib.EqualFoldRight(ffj_key_ChangeEvent_Service, kn) {
					currentKey = ffj_t_ChangeEvent_Service
					state = fflib.FFParse_want_colon
					goto mainparse
				}

				currentKey = ffj_t_ChangeEventno_such_key
				state = fflib.FFParse_want_colon
				goto mainparse
			}

		case fflib.FFParse_want_colon:
			if tok != fflib.FFTok_colon {
				wantedTok = fflib.FFTok_colon
				goto wrongtokenerror
			}
			state = fflib.FFParse_want_value
			continue
		case fflib.FFParse_want_value:

			if tok == fflib.FFTok_left_brace || tok == fflib.FFTok_left_bracket || tok == fflib.FFTok_integer || tok == fflib.FFTok_double || tok == fflib.FFTok_string || tok == fflib.FFTok_bool || tok == fflib.FFTok_null {
				switch currentKey {

				case ffj_t_ChangeEvent_Service:
					goto handle_Service

				case ffj_t_ChangeEvent_PreviousStatus:
					goto handle_PreviousStatus

				case ffj_t_ChangeEvent_Time:
					goto handle_Time

				case ffj_t_ChangeEventno_such_key:
					err = fs.SkipField(tok)
					if err != nil {
						return fs.WrapErr(err)
					}
					state = fflib.FFParse_after_value
					goto mainparse
				}
			} else {
				goto wantedvalue
			}
		}
	}

handle_Service:

	/* handler: uj.Service type=service.Service kind=struct quoted=false*/

	{
		if tok == fflib.FFTok_null {

			state = fflib.FFParse_after_value
			goto mainparse
		}

		err = uj.Service.UnmarshalJSONFFLexer(fs, fflib.FFParse_want_key)
		if err != nil {
			return err
		}
		state = fflib.FFParse_after_value
	}

	state = fflib.FFParse_after_value
	goto mainparse

handle_PreviousStatus:

	/* handler: uj.PreviousStatus type=int kind=int quoted=false*/

	{
		if tok != fflib.FFTok_integer && tok != fflib.FFTok_null {
			return fs.WrapErr(fmt.Errorf("cannot unmarshal %s into Go value for int", tok))
		}
	}

	{

		if tok == fflib.FFTok_null {

		} else {

			tval, err := fflib.ParseInt(fs.Output.Bytes(), 10, 64)

			if err != nil {
				return fs.WrapErr(err)
			}

			uj.PreviousStatus = int(tval)

		}
	}

	state = fflib.FFParse_after_value
	goto mainparse

handle_Time:

	/* handler: uj.Time type=time.Time kind=struct quoted=false*/

	{
		if tok == fflib.FFTok_null {

			state = fflib.FFParse_after_value
			goto mainparse
		}

		tbuf, err := fs.CaptureField(tok)
		if err != nil {
			return fs.WrapErr(err)
		}

		err = uj.Time.UnmarshalJSON(tbuf)
		if err != nil {
			return fs.WrapErr(err)
		}
		state = fflib.FFParse_after_value
	}

	state = fflib.FFParse_after_value
	goto mainparse

wantedvalue:
	return fs.WrapErr(fmt.Errorf("wanted value token, but got token: %v", tok))
wrongtokenerror:
	return fs.WrapErr(fmt.Errorf("ffjson: wanted token: %v, but got token: %v output=%s", wantedTok, tok, fs.Output.String()))
tokerror:
	if fs.BigError != nil {
		return fs.WrapErr(fs.BigError)
	}
	err = fs.Error.ToError()
	if err != nil {
		return fs.WrapErr(err)
	}
	panic("ffjson-generated: unreachable, please report bug.")
done:

	return nil
}

func (mj *Server) MarshalJSON() ([]byte, error) {
	var buf fflib.Buffer
	if mj == nil {
		buf.WriteString("null")
		return buf.Bytes(), nil
	}
	err := mj.MarshalJSONBuf(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
func (mj *Server) MarshalJSONBuf(buf fflib.EncodingBuffer) error {
	if mj == nil {
		buf.WriteString("null")
		return nil
	}
	var err error
	var obj []byte
	_ = obj
	_ = err
	buf.WriteString(`{"Name":`)
	fflib.WriteJsonString(buf, string(mj.Name))
	buf.WriteString(`,"Services":`)
	/* Falling back. type=map[string]*service.Service kind=map */
	err = buf.Encode(mj.Services)
	if err != nil {
		return err
	}
	buf.WriteString(`,"LastUpdated":`)

	{

		obj, err = mj.LastUpdated.MarshalJSON()
		if err != nil {
			return err
		}
		buf.Write(obj)

	}
	buf.WriteString(`,"LastChanged":`)

	{

		obj, err = mj.LastChanged.MarshalJSON()
		if err != nil {
			return err
		}
		buf.Write(obj)

	}
	buf.WriteByte('}')
	return nil
}

const (
	ffj_t_Serverbase = iota
	ffj_t_Serverno_such_key

	ffj_t_Server_Name

	ffj_t_Server_Services

	ffj_t_Server_LastUpdated

	ffj_t_Server_LastChanged
)

var ffj_key_Server_Name = []byte("Name")

var ffj_key_Server_Services = []byte("Services")

var ffj_key_Server_LastUpdated = []byte("LastUpdated")

var ffj_key_Server_LastChanged = []byte("LastChanged")

func (uj *Server) UnmarshalJSON(input []byte) error {
	fs := fflib.NewFFLexer(input)
	return uj.UnmarshalJSONFFLexer(fs, fflib.FFParse_map_start)
}

func (uj *Server) UnmarshalJSONFFLexer(fs *fflib.FFLexer, state fflib.FFParseState) error {
	var err error = nil
	currentKey := ffj_t_Serverbase
	_ = currentKey
	tok := fflib.FFTok_init
	wantedTok := fflib.FFTok_init

mainparse:
	for {
		tok = fs.Scan()
		//	println(fmt.Sprintf("debug: tok: %v  state: %v", tok, state))
		if tok == fflib.FFTok_error {
			goto tokerror
		}

		switch state {

		case fflib.FFParse_map_start:
			if tok != fflib.FFTok_left_bracket {
				wantedTok = fflib.FFTok_left_bracket
				goto wrongtokenerror
			}
			state = fflib.FFParse_want_key
			continue

		case fflib.FFParse_after_value:
			if tok == fflib.FFTok_comma {
				state = fflib.FFParse_want_key
			} else if tok == fflib.FFTok_right_bracket {
				goto done
			} else {
				wantedTok = fflib.FFTok_comma
				goto wrongtokenerror
			}

		case fflib.FFParse_want_key:
			// json {} ended. goto exit. woo.
			if tok == fflib.FFTok_right_bracket {
				goto done
			}
			if tok != fflib.FFTok_string {
				wantedTok = fflib.FFTok_string
				goto wrongtokenerror
			}

			kn := fs.Output.Bytes()
			if len(kn) <= 0 {
				// "" case. hrm.
				currentKey = ffj_t_Serverno_such_key
				state = fflib.FFParse_want_colon
				goto mainparse
			} else {
				switch kn[0] {

				case 'L':

					if bytes.Equal(ffj_key_Server_LastUpdated, kn) {
						currentKey = ffj_t_Server_LastUpdated
						state = fflib.FFParse_want_colon
						goto mainparse

					} else if bytes.Equal(ffj_key_Server_LastChanged, kn) {
						currentKey = ffj_t_Server_LastChanged
						state = fflib.FFParse_want_colon
						goto mainparse
					}

				case 'N':

					if bytes.Equal(ffj_key_Server_Name, kn) {
						currentKey = ffj_t_Server_Name
						state = fflib.FFParse_want_colon
						goto mainparse
					}

				case 'S':

					if bytes.Equal(ffj_key_Server_Services, kn) {
						currentKey = ffj_t_Server_Services
						state = fflib.FFParse_want_colon
						goto mainparse
					}

				}

				if fflib.EqualFoldRight(ffj_key_Server_LastChanged, kn) {
					currentKey = ffj_t_Server_LastChanged
					state = fflib.FFParse_want_colon
					goto mainparse
				}

				if fflib.EqualFoldRight(ffj_key_Server_LastUpdated, kn) {
					currentKey = ffj_t_Server_LastUpdated
					state = fflib.FFParse_want_colon
					goto mainparse
				}

				if fflib.EqualFoldRight(ffj_key_Server_Services, kn) {
					currentKey = ffj_t_Server_Services
					state = fflib.FFParse_want_colon
					goto mainparse
				}

				if fflib.SimpleLetterEqualFold(ffj_key_Server_Name, kn) {
					currentKey = ffj_t_Server_Name
					state = fflib.FFParse_want_colon
					goto mainparse
				}

				currentKey = ffj_t_Serverno_such_key
				state = fflib.FFParse_want_colon
				goto mainparse
			}

		case fflib.FFParse_want_colon:
			if tok != fflib.FFTok_colon {
				wantedTok = fflib.FFTok_colon
				goto wrongtokenerror
			}
			state = fflib.FFParse_want_value
			continue
		case fflib.FFParse_want_value:

			if tok == fflib.FFTok_left_brace || tok == fflib.FFTok_left_bracket || tok == fflib.FFTok_integer || tok == fflib.FFTok_double || tok == fflib.FFTok_string || tok == fflib.FFTok_bool || tok == fflib.FFTok_null {
				switch currentKey {

				case ffj_t_Server_Name:
					goto handle_Name

				case ffj_t_Server_Services:
					goto handle_Services

				case ffj_t_Server_LastUpdated:
					goto handle_LastUpdated

				case ffj_t_Server_LastChanged:
					goto handle_LastChanged

				case ffj_t_Serverno_such_key:
					err = fs.SkipField(tok)
					if err != nil {
						return fs.WrapErr(err)
					}
					state = fflib.FFParse_after_value
					goto mainparse
				}
			} else {
				goto wantedvalue
			}
		}
	}

handle_Name:

	/* handler: uj.Name type=string kind=string quoted=false*/

	{

		{
			if tok != fflib.FFTok_string && tok != fflib.FFTok_null {
				return fs.WrapErr(fmt.Errorf("cannot unmarshal %s into Go value for string", tok))
			}
		}

		if tok == fflib.FFTok_null {

		} else {

			outBuf := fs.Output.Bytes()

			uj.Name = string(string(outBuf))

		}
	}

	state = fflib.FFParse_after_value
	goto mainparse

handle_Services:

	/* handler: uj.Services type=map[string]*service.Service kind=map quoted=false*/

	{

		{
			if tok != fflib.FFTok_left_bracket && tok != fflib.FFTok_null {
				return fs.WrapErr(fmt.Errorf("cannot unmarshal %s into Go value for ", tok))
			}
		}

		if tok == fflib.FFTok_null {
			uj.Services = nil
		} else {

			uj.Services = make(map[string]*service.Service, 0)

			wantVal := true

			for {

				var k string

				var tmp_uj__Services *service.Service

				tok = fs.Scan()
				if tok == fflib.FFTok_error {
					goto tokerror
				}
				if tok == fflib.FFTok_right_bracket {
					break
				}

				if tok == fflib.FFTok_comma {
					if wantVal == true {
						// TODO(pquerna): this isn't an ideal error message, this handles
						// things like [,,,] as an array value.
						return fs.WrapErr(fmt.Errorf("wanted value token, but got token: %v", tok))
					}
					continue
				} else {
					wantVal = true
				}

				/* handler: k type=string kind=string quoted=false*/

				{

					{
						if tok != fflib.FFTok_string && tok != fflib.FFTok_null {
							return fs.WrapErr(fmt.Errorf("cannot unmarshal %s into Go value for string", tok))
						}
					}

					if tok == fflib.FFTok_null {

					} else {

						outBuf := fs.Output.Bytes()

						k = string(string(outBuf))

					}
				}

				// Expect ':' after key
				tok = fs.Scan()
				if tok != fflib.FFTok_colon {
					return fs.WrapErr(fmt.Errorf("wanted colon token, but got token: %v", tok))
				}

				tok = fs.Scan()
				/* handler: tmp_uj__Services type=*service.Service kind=ptr quoted=false*/

				{
					if tok == fflib.FFTok_null {

						tmp_uj__Services = nil

						state = fflib.FFParse_after_value
						goto mainparse
					}

					if tmp_uj__Services == nil {
						tmp_uj__Services = new(service.Service)
					}

					err = tmp_uj__Services.UnmarshalJSONFFLexer(fs, fflib.FFParse_want_key)
					if err != nil {
						return err
					}
					state = fflib.FFParse_after_value
				}

				uj.Services[k] = tmp_uj__Services

				wantVal = false
			}

		}
	}

	state = fflib.FFParse_after_value
	goto mainparse

handle_LastUpdated:

	/* handler: uj.LastUpdated type=time.Time kind=struct quoted=false*/

	{
		if tok == fflib.FFTok_null {

			state = fflib.FFParse_after_value
			goto mainparse
		}

		tbuf, err := fs.CaptureField(tok)
		if err != nil {
			return fs.WrapErr(err)
		}

		err = uj.LastUpdated.UnmarshalJSON(tbuf)
		if err != nil {
			return fs.WrapErr(err)
		}
		state = fflib.FFParse_after_value
	}

	state = fflib.FFParse_after_value
	goto mainparse

handle_LastChanged:

	/* handler: uj.LastChanged type=time.Time kind=struct quoted=false*/

	{
		if tok == fflib.FFTok_null {

			state = fflib.FFParse_after_value
			goto mainparse
		}

		tbuf, err := fs.CaptureField(tok)
		if err != nil {
			return fs.WrapErr(err)
		}

		err = uj.LastChanged.UnmarshalJSON(tbuf)
		if err != nil {
			return fs.WrapErr(err)
		}
		state = fflib.FFParse_after_value
	}

	state = fflib.FFParse_after_value
	goto mainparse

wantedvalue:
	return fs.WrapErr(fmt.Errorf("wanted value token, but got token: %v", tok))
wrongtokenerror:
	return fs.WrapErr(fmt.Errorf("ffjson: wanted token: %v, but got token: %v output=%s", wantedTok, tok, fs.Output.String()))
tokerror:
	if fs.BigError != nil {
		return fs.WrapErr(fs.BigError)
	}
	err = fs.Error.ToError()
	if err != nil {
		return fs.WrapErr(err)
	}
	panic("ffjson-generated: unreachable, please report bug.")
done:

	return nil
}

func (mj *ServicesState) MarshalJSON() ([]byte, error) {
	var buf fflib.Buffer
	if mj == nil {
		buf.WriteString("null")
		return buf.Bytes(), nil
	}
	err := mj.MarshalJSONBuf(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
func (mj *ServicesState) MarshalJSONBuf(buf fflib.EncodingBuffer) error {
	if mj == nil {
		buf.WriteString("null")
		return nil
	}
	var err error
	var obj []byte
	_ = obj
	_ = err
	buf.WriteString(`{"Servers":`)
	/* Falling back. type=map[string]*catalog.Server kind=map */
	err = buf.Encode(mj.Servers)
	if err != nil {
		return err
	}
	buf.WriteString(`,"LastChanged":`)

	{

		obj, err = mj.LastChanged.MarshalJSON()
		if err != nil {
			return err
		}
		buf.Write(obj)

	}
	buf.WriteString(`,"ClusterName":`)
	fflib.WriteJsonString(buf, string(mj.ClusterName))
	buf.WriteString(`,"Hostname":`)
	fflib.WriteJsonString(buf, string(mj.Hostname))
	buf.WriteByte('}')
	return nil
}

const (
	ffj_t_ServicesStatebase = iota
	ffj_t_ServicesStateno_such_key

	ffj_t_ServicesState_Servers

	ffj_t_ServicesState_LastChanged

	ffj_t_ServicesState_ClusterName

	ffj_t_ServicesState_Hostname
)

var ffj_key_ServicesState_Servers = []byte("Servers")

var ffj_key_ServicesState_LastChanged = []byte("LastChanged")

var ffj_key_ServicesState_ClusterName = []byte("ClusterName")

var ffj_key_ServicesState_Hostname = []byte("Hostname")

func (uj *ServicesState) UnmarshalJSON(input []byte) error {
	fs := fflib.NewFFLexer(input)
	return uj.UnmarshalJSONFFLexer(fs, fflib.FFParse_map_start)
}

func (uj *ServicesState) UnmarshalJSONFFLexer(fs *fflib.FFLexer, state fflib.FFParseState) error {
	var err error = nil
	currentKey := ffj_t_ServicesStatebase
	_ = currentKey
	tok := fflib.FFTok_init
	wantedTok := fflib.FFTok_init

mainparse:
	for {
		tok = fs.Scan()
		//	println(fmt.Sprintf("debug: tok: %v  state: %v", tok, state))
		if tok == fflib.FFTok_error {
			goto tokerror
		}

		switch state {

		case fflib.FFParse_map_start:
			if tok != fflib.FFTok_left_bracket {
				wantedTok = fflib.FFTok_left_bracket
				goto wrongtokenerror
			}
			state = fflib.FFParse_want_key
			continue

		case fflib.FFParse_after_value:
			if tok == fflib.FFTok_comma {
				state = fflib.FFParse_want_key
			} else if tok == fflib.FFTok_right_bracket {
				goto done
			} else {
				wantedTok = fflib.FFTok_comma
				goto wrongtokenerror
			}

		case fflib.FFParse_want_key:
			// json {} ended. goto exit. woo.
			if tok == fflib.FFTok_right_bracket {
				goto done
			}
			if tok != fflib.FFTok_string {
				wantedTok = fflib.FFTok_string
				goto wrongtokenerror
			}

			kn := fs.Output.Bytes()
			if len(kn) <= 0 {
				// "" case. hrm.
				currentKey = ffj_t_ServicesStateno_such_key
				state = fflib.FFParse_want_colon
				goto mainparse
			} else {
				switch kn[0] {

				case 'C':

					if bytes.Equal(ffj_key_ServicesState_ClusterName, kn) {
						currentKey = ffj_t_ServicesState_ClusterName
						state = fflib.FFParse_want_colon
						goto mainparse
					}

				case 'H':

					if bytes.Equal(ffj_key_ServicesState_Hostname, kn) {
						currentKey = ffj_t_ServicesState_Hostname
						state = fflib.FFParse_want_colon
						goto mainparse
					}

				case 'L':

					if bytes.Equal(ffj_key_ServicesState_LastChanged, kn) {
						currentKey = ffj_t_ServicesState_LastChanged
						state = fflib.FFParse_want_colon
						goto mainparse
					}

				case 'S':

					if bytes.Equal(ffj_key_ServicesState_Servers, kn) {
						currentKey = ffj_t_ServicesState_Servers
						state = fflib.FFParse_want_colon
						goto mainparse
					}

				}

				if fflib.EqualFoldRight(ffj_key_ServicesState_Hostname, kn) {
					currentKey = ffj_t_ServicesState_Hostname
					state = fflib.FFParse_want_colon
					goto mainparse
				}

				if fflib.EqualFoldRight(ffj_key_ServicesState_ClusterName, kn) {
					currentKey = ffj_t_ServicesState_ClusterName
					state = fflib.FFParse_want_colon
					goto mainparse
				}

				if fflib.EqualFoldRight(ffj_key_ServicesState_LastChanged, kn) {
					currentKey = ffj_t_ServicesState_LastChanged
					state = fflib.FFParse_want_colon
					goto mainparse
				}

				if fflib.EqualFoldRight(ffj_key_ServicesState_Servers, kn) {
					currentKey = ffj_t_ServicesState_Servers
					state = fflib.FFParse_want_colon
					goto mainparse
				}

				currentKey = ffj_t_ServicesStateno_such_key
				state = fflib.FFParse_want_colon
				goto mainparse
			}

		case fflib.FFParse_want_colon:
			if tok != fflib.FFTok_colon {
				wantedTok = fflib.FFTok_colon
				goto wrongtokenerror
			}
			state = fflib.FFParse_want_value
			continue
		case fflib.FFParse_want_value:

			if tok == fflib.FFTok_left_brace || tok == fflib.FFTok_left_bracket || tok == fflib.FFTok_integer || tok == fflib.FFTok_double || tok == fflib.FFTok_string || tok == fflib.FFTok_bool || tok == fflib.FFTok_null {
				switch currentKey {

				case ffj_t_ServicesState_Servers:
					goto handle_Servers

				case ffj_t_ServicesState_LastChanged:
					goto handle_LastChanged

				case ffj_t_ServicesState_ClusterName:
					goto handle_ClusterName

				case ffj_t_ServicesState_Hostname:
					goto handle_Hostname

				case ffj_t_ServicesStateno_such_key:
					err = fs.SkipField(tok)
					if err != nil {
						return fs.WrapErr(err)
					}
					state = fflib.FFParse_after_value
					goto mainparse
				}
			} else {
				goto wantedvalue
			}
		}
	}

handle_Servers:

	/* handler: uj.Servers type=map[string]*catalog.Server kind=map quoted=false*/

	{

		{
			if tok != fflib.FFTok_left_bracket && tok != fflib.FFTok_null {
				return fs.WrapErr(fmt.Errorf("cannot unmarshal %s into Go value for ", tok))
			}
		}

		if tok == fflib.FFTok_null {
			uj.Servers = nil
		} else {

			uj.Servers = make(map[string]*Server, 0)

			wantVal := true

			for {

				var k string

				var tmp_uj__Servers *Server

				tok = fs.Scan()
				if tok == fflib.FFTok_error {
					goto tokerror
				}
				if tok == fflib.FFTok_right_bracket {
					break
				}

				if tok == fflib.FFTok_comma {
					if wantVal == true {
						// TODO(pquerna): this isn't an ideal error message, this handles
						// things like [,,,] as an array value.
						return fs.WrapErr(fmt.Errorf("wanted value token, but got token: %v", tok))
					}
					continue
				} else {
					wantVal = true
				}

				/* handler: k type=string kind=string quoted=false*/

				{

					{
						if tok != fflib.FFTok_string && tok != fflib.FFTok_null {
							return fs.WrapErr(fmt.Errorf("cannot unmarshal %s into Go value for string", tok))
						}
					}

					if tok == fflib.FFTok_null {

					} else {

						outBuf := fs.Output.Bytes()

						k = string(string(outBuf))

					}
				}

				// Expect ':' after key
				tok = fs.Scan()
				if tok != fflib.FFTok_colon {
					return fs.WrapErr(fmt.Errorf("wanted colon token, but got token: %v", tok))
				}

				tok = fs.Scan()
				/* handler: tmp_uj__Servers type=*catalog.Server kind=ptr quoted=false*/

				{
					if tok == fflib.FFTok_null {

						tmp_uj__Servers = nil

						state = fflib.FFParse_after_value
						goto mainparse
					}

					if tmp_uj__Servers == nil {
						tmp_uj__Servers = new(Server)
					}

					err = tmp_uj__Servers.UnmarshalJSONFFLexer(fs, fflib.FFParse_want_key)
					if err != nil {
						return err
					}
					state = fflib.FFParse_after_value
				}

				uj.Servers[k] = tmp_uj__Servers

				wantVal = false
			}

		}
	}

	state = fflib.FFParse_after_value
	goto mainparse

handle_LastChanged:

	/* handler: uj.LastChanged type=time.Time kind=struct quoted=false*/

	{
		if tok == fflib.FFTok_null {

			state = fflib.FFParse_after_value
			goto mainparse
		}

		tbuf, err := fs.CaptureField(tok)
		if err != nil {
			return fs.WrapErr(err)
		}

		err = uj.LastChanged.UnmarshalJSON(tbuf)
		if err != nil {
			return fs.WrapErr(err)
		}
		state = fflib.FFParse_after_value
	}

	state = fflib.FFParse_after_value
	goto mainparse

handle_ClusterName:

	/* handler: uj.ClusterName type=string kind=string quoted=false*/

	{

		{
			if tok != fflib.FFTok_string && tok != fflib.FFTok_null {
				return fs.WrapErr(fmt.Errorf("cannot unmarshal %s into Go value for string", tok))
			}
		}

		if tok == fflib.FFTok_null {

		} else {

			outBuf := fs.Output.Bytes()

			uj.ClusterName = string(string(outBuf))

		}
	}

	state = fflib.FFParse_after_value
	goto mainparse

handle_Hostname:

	/* handler: uj.Hostname type=string kind=string quoted=false*/

	{

		{
			if tok != fflib.FFTok_string && tok != fflib.FFTok_null {
				return fs.WrapErr(fmt.Errorf("cannot unmarshal %s into Go value for string", tok))
			}
		}

		if tok == fflib.FFTok_null {

		} else {

			outBuf := fs.Output.Bytes()

			uj.Hostname = string(string(outBuf))

		}
	}

	state = fflib.FFParse_after_value
	goto mainparse

wantedvalue:
	return fs.WrapErr(fmt.Errorf("wanted value token, but got token: %v", tok))
wrongtokenerror:
	return fs.WrapErr(fmt.Errorf("ffjson: wanted token: %v, but got token: %v output=%s", wantedTok, tok, fs.Output.String()))
tokerror:
	if fs.BigError != nil {
		return fs.WrapErr(fs.BigError)
	}
	err = fs.Error.ToError()
	if err != nil {
		return fs.WrapErr(err)
	}
	panic("ffjson-generated: unreachable, please report bug.")
done:

	return nil
}
