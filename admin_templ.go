// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.747
package main

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

func Admin() templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<!doctype html><html><head><title>Admin - Vote Records</title><style>\n\t\t\t\tbody { font-family: Arial, sans-serif; margin: 0; padding: 20px; }\n\t\t\t\ttable { width: 100%; border-collapse: collapse; }\n\t\t\t\tth, td { border: 1px solid #ddd; padding: 8px; text-align: left; }\n\t\t\t\tth { background-color: #f2f2f2; }\n\t\t\t</style></head><body><h1>Admin - Vote Records</h1><div id=\"vote-records\"></div><script>\n\t\t\t\tconst socket = new WebSocket(\"ws://\" + window.location.host + \"/admin/ws\");\n\n\t\t\t\tsocket.onmessage = function(event) {\n\t\t\t\t\tconsole.log('Received WebSocket message:', event.data);\n\t\t\t\t\ttry {\n\t\t\t\t\t\tconst data = JSON.parse(event.data);\n\t\t\t\t\t\tif (Array.isArray(data)) {\n\t\t\t\t\t\t\tupdateVoteRecords(data);\n\t\t\t\t\t\t} else {\n\t\t\t\t\t\t\tconsole.error('Received data is not an array:', data);\n\t\t\t\t\t\t}\n\t\t\t\t\t} catch (error) {\n\t\t\t\t\t\tconsole.error('Error parsing WebSocket data:', error);\n\t\t\t\t\t}\n\t\t\t\t};\n\n\t\t\t\tfunction updateVoteRecords(data) {\n\t\t\t\t\tconst container = document.getElementById(\"vote-records\");\n\t\t\t\t\tlet html = '<table><tr><th>Round</th><th>User ID</th><th>Vote</th></tr>';\n\n\t\t\t\t\tdata.forEach(record => {\n\t\t\t\t\t\thtml += `<tr>\n\t\t\t\t\t\t\t<td>${record.roundName}</td>\n\t\t\t\t\t\t\t<td>${record.userId}</td>\n\t\t\t\t\t\t\t<td>${record.voteName}</td>\n\t\t\t\t\t\t</tr>`;\n\t\t\t\t\t});\n\n\t\t\t\t\thtml += '</table>';\n\t\t\t\t\tcontainer.innerHTML = html;\n\t\t\t\t}\n\n\t\t\t\t// Initial request for vote records\n\t\t\t\tfetch(\"/admin/vote-records\")\n\t\t\t\t\t.then(response => response.json())\n\t\t\t\t\t.then(data => {\n\t\t\t\t\t\tconsole.log('Initial vote records:', data);\n\t\t\t\t\t\tupdateVoteRecords(data);\n\t\t\t\t\t})\n\t\t\t\t\t.catch(error => console.error('Error fetching initial vote records:', error));\n\t\t\t</script></body></html>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}
