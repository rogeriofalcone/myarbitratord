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

// member variables that start with capital letters are public/exported 
type Instance struct {
  Mysql_host string 
  Mysql_port string
  Mysql_user string
  mysql_pass string

  // The status related vars can serve as an effective cache 
  Group_name string
  Server_uuid string
  Member_state string
  Online_participants int
  Has_quorum bool
  Read_only bool
  Applier_queue_size uint16
  db *sql.DB
}


func New( myh string, myp string, myu string, mys string ) * Instance {
  return &Instance{ Mysql_host: myh, Mysql_port: myp, Mysql_user: myu, mysql_pass: mys }
}

func (me *Instance) Connect() error {
  var err error 
  me.db, err = sql.Open("mysql", me.Mysql_user + ":" + me.mysql_pass + "@tcp(" + me.Mysql_host + ":" + me.Mysql_port + ")/performance_schema")

  if( err == nil ){
    err = me.db.Ping()
    if( err == nil ){
      //defer me.db.Close()

      err = me.db.QueryRow( "SELECT variable_value FROM global_variables WHERE variable_name='group_replication_group_name'" ).Scan( &me.Group_name )

      if( err != nil || me.Group_name == "" ){
        err = errors.New( "Specified MySQL Instance is not a member of any Group Replication cluster!" )
      }

      err = me.db.QueryRow( "SELECT variable_value, member_state FROM global_variables gv INNER JOIN replication_group_members rgm ON(gv.variable_value=rgm.member_state) WHERE gv.variable_name='server_uuid'" ).Scan( &me.Server_uuid, &me.Member_state )
    }
  }
  
  return err
}

func (me *Instance) HasQuorum() (bool, error) {
  quorum_query := "SELECT IF( MEMBER_STATE='ONLINE' AND ((SELECT COUNT(*) FROM replication_group_members WHERE MEMBER_STATE != 'ONLINE') >= ((SELECT COUNT(*) FROM replication_group_members)/2) = 0), 'true', 'false' ) FROM replication_group_members JOIN replication_group_member_stats USING(member_id)"

  err := me.db.QueryRow( quorum_query ).Scan( &me.Has_quorum )
 
  return me.Has_quorum, err
}

func (me *Instance) IsReadOnly() (bool, error) {
  ro_query := "SELECT variable_value FROM global_variables WHERE variable_name='super_read_only'"
  err := me.db.QueryRow( ro_query ).Scan( &me.Read_only )

  return me.Read_only, err
}

func (me *Instance) GetMembers() (*[]Instance, error) {
  membership_query := "SELECT member_id, member_host, member_port, member_state FROM replication_group_members"
  member_slice := []Instance{}
  Online_participants := 0

  rows, err := me.db.Query( membership_query )

  if( err == nil ){
    defer rows.Close()

    for( rows.Next() ){
      member := New( "", "", "", "")
      err = rows.Scan( member.Server_uuid, member.Mysql_host, member.Mysql_port, member.Member_state )
      if( member.Member_state == "ONLINE" ){
        Online_participants++ 
      }
      member_slice = append( member_slice, *member )
    }
  }

  return &member_slice, err 
}

func (me *Instance) Shutdown() error {
  shutdown_query := "SHUTDOWN"

  _, err := me.db.Exec( shutdown_query )

  return err
}

func (me *Instance) ForceMembers( fms string ) error {
  force_membership_query := "SET GLOBAL group_replication_force_members='" + fms + "'"

  _, err := me.db.Exec( force_membership_query )

  return err
}
 

