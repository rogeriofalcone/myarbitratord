/*
Copyright 2017 Matthew Lord (mattalord@gmail.com) 

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package instance

import (
  "database/sql"
  _ "github.com/go-sql-driver/mysql"
  "errors"
)

type instance struct {
  mysql_host string 
  mysql_port string
  mysql_user string
  mysql_pass string

  group_name string
  group_status string 
  has_quorum bool
  applier_queue_size uint16
}

var db *sql.DB

func New( myh string, myp string, myu string, mys string ) * instance {
  return &instance{ mysql_host: myh, mysql_port: myp, mysql_user: myu, mysql_pass: mys }
}

func (me *instance) Connect() error {
  var err error 
  db, err = sql.Open("mysql", me.mysql_user + ":" + me.mysql_pass + "@tcp(" + me.mysql_host + ":" + me.mysql_port + ")/performance_schema")

  if( err == nil ){
    err = db.Ping()
    if( err == nil ){
      //defer db.Close()

      err = db.QueryRow( "select variable_value from global_variables where variable_name='group_replication_group_name'" ).Scan( &me.group_name )

      if( err != nil || me.group_name == "" ){
        err = errors.New( "Specified MySQL instance is not a member of any Group Replication cluster!" )
      }
    }
  }
  
  return err
}

func (me *instance) HasQuorum() (error, bool) {
  quorum_query := "SELECT IF( MEMBER_STATE='ONLINE' AND ((SELECT COUNT(*) FROM performance_schema.replication_group_members WHERE MEMBER_STATE != 'ONLINE') >= ((SELECT COUNT(*) FROM performance_schema.replication_group_members)/2) = 0), 'true', 'false' ) FROM performance_schema.replication_group_members JOIN performance_schema.replication_group_member_stats USING(member_id)"

  err := db.QueryRow( quorum_query ).Scan( &me.has_quorum )
 
  return err, me.has_quorum
}

