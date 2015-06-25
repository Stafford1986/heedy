package authoperator

import "connectordb/streamdb/users"

//ReadAllUsers reads all the users
func (o *AuthOperator) ReadAllUsers() ([]users.User, error) {
	if o.Permissions(users.ROOT) {
		return o.Db.ReadAllUsers()
	}
	//If not admin, then we only know about our own user
	u, err := o.User()
	if err != nil {
		return []users.User{}, err
	}
	return []users.User{*u}, err
}

//CreateUser makes a new user
func (o *AuthOperator) CreateUser(username, email, password string) error {
	if o.Permissions(users.ROOT) {
		err := o.Db.CreateUser(username, email, password)
		if err == nil {
			o.UserLog("CreateUser", username)
		}
		return err
	}
	return ErrPermissions
}

//ReadUser reads a user - or rather reads any user that this device has permissions to read
func (o *AuthOperator) ReadUser(username string) (*users.User, error) {
	if o.Permissions(users.ROOT) {
		return o.Db.ReadUser(username)
	}
	//Not an admin. See if it is asking about the current user
	if u, err := o.User(); err == nil && u.Name == username {
		return u, nil
	}
	return nil, ErrPermissions
}

//ReadUserByID reads the user given the ID
func (o *AuthOperator) ReadUserByID(userID int64) (*users.User, error) {
	if o.Permissions(users.ROOT) {
		return o.Db.ReadUserByID(userID)
	}
	if usr, err := o.User(); err == nil && usr.UserId == userID {
		return usr, nil
	}
	return nil, ErrPermissions
}

//ReadUserByEmail reads a user - or rather reads any user that this device has permissions to read
func (o *AuthOperator) ReadUserByEmail(email string) (*users.User, error) {
	if o.Permissions(users.ROOT) {
		return o.Db.ReadUserByEmail(email)
	}
	if u, err := o.User(); err == nil && u.Email == email {
		return u, nil
	}
	return nil, ErrPermissions
}

//UpdateUser performs the given modifications
func (o *AuthOperator) UpdateUser(modifieduser *users.User) error {
	user, err := o.ReadUserByID(modifieduser.UserId)
	if err != nil {
		return err
	}
	dev, err := o.Device()
	if err != nil {
		return err
	}

	//See if the bastards tried to change a field they have no fucking business editing :-P
	if modifieduser.RevertUneditableFields(*user, dev.RelationToUser(user)) > 0 {
		return ErrPermissions
	}
	//Thankfully, ReadUser put this user right on top of the cache, so it should still be there
	err = o.Db.UpdateUser(modifieduser)
	if err == nil {
		o.UserLog("UpdateUser", modifieduser.Name)
	}
	return err
}

//DeleteUserByID deletes the given user - only admin can delete
func (o *AuthOperator) DeleteUserByID(userID int64) error {
	if o.Permissions(users.ROOT) {
		usr, err1 := o.ReadUserByID(userID)

		err := o.Db.DeleteUserByID(userID)
		if err == nil && err1 == nil {
			o.UserLog("DeleteUser", usr.Name)
		}
		return err
	}
	return ErrPermissions

}
