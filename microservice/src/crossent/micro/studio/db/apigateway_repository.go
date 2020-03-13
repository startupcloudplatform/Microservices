package db

import (
    "crossent/micro/studio/db/lock"
    "crossent/micro/studio/domain"
    "database/sql"
    "fmt"
    sq "github.com/Masterminds/squirrel"
    "github.com/pkg/errors"
    "time"
)



type ApigatewayRepository interface {
    ID() int
    ListMicroserviceApi(int, string, string) ([]domain.MicroApi, error)
    GetMicroserviceApiRule() (domain.MicroApiRule, error)
    SaveMicroserviceApi(domain.MicroApi) error
    ListMicroserviceFrontend([]string) ([]domain.View, error)
    GetMicroserviceApi(int) (domain.MicroApi, error)
    DeleteMicroserviceApi(domain.MicroApi, string) error
    DeleteMicroserviceAppApi(int, int, string) error
    ListMicroserviceAppApiByMicroID(int) ([]domain.MicroApi, error)
    CreateMicroserviceAppApi(domain.MicroApi) (bool, error)
    SaveMicroserviceApiRule(domain.MicroApi) error
    GetMicroserviceNameCheck(string) (bool, error)
    GetMicroservicePathCheck(string) (bool, error)
    SaveMicroserviceSwaggerApi(domain.View) error

    GetMicroserviceAppApi(int, int) (domain.MicroAppApi, error)
    ListMicroserviceAppApiByApiID(int) ([]domain.MicroApi, error)
}

type apigatewayRepository struct {
    id          int
    conn        Conn
    lockFactory lock.LockFactory
    name        string
}

func (a *apigatewayRepository) ID() int { return a.id }

func newApigatewayRepository(conn Conn, lockFactory lock.LockFactory) *apigatewayRepository {
    return &apigatewayRepository{
        conn:        conn,
        lockFactory: lockFactory,
    }
}



func (a *apigatewayRepository) ListMicroserviceApi(offset int, name string, orgguid string) ([]domain.MicroApi, error) {
    //name_condition := sq.Eq{"1": "1"}
    //user_condition := sq.Eq{"1": "1"}
    //if name != "" {
    //	name_condition = sq.Eq{"name": name}
    //}
    //if userid != "" {
    //	user_condition = sq.Eq{"user_id": userid}
    //}
    //rows, err := psql.Select(`
    //		a.id,
    //		a.part,
    //		a.name,
    //		a.host,
    //		a.path,
    //		a.version,
    //		a.rest_api,
    //		a.active,
    //		a.description,
    //		a.image,
    //		a.user_id,
    //		a.updated
    //	`).
    //	From("micro_api a").
    //	Where(name_condition).
    //	Where(user_condition).
    //	Where(sq.Eq{"active": "Y"}).
    //	OrderBy("updated DESC").
    //	Offset(uint64(offset)).
    //	Limit(6).
    //	RunWith(a.conn).
    //	Query()
    wherename := fmt.Sprintf("%s%%", name)
    whereorg := fmt.Sprintf("%s%%", orgguid)
    rows, err := a.conn.Query(`SELECT a.id, a.part, a.name, a.host, a.path, a.version, a.rest_api, a.active, a.description, a.image
                                      ,a.user_id, a.updated, COALESCE(b.favorite, 0) favorite
                               FROM micro_api a
                               LEFT OUTER JOIN (SELECT api_id, count(id) favorite FROM micro_app_api WHERE active='Y' GROUP BY api_id) b
                               ON a.id = b.api_id
                               WHERE a.active='Y'
                               AND a.org_guid LIKE $1
                               AND a.name LIKE $2
                               ORDER BY a.updated DESC
                               OFFSET $3
                               LIMIT 6`, whereorg, wherename, offset )

    if err != nil {
        return nil, err
    }

    defer rows.Close()

    views := []domain.MicroApi{}
    for rows.Next() {
        view := domain.MicroApi{}


        err := rows.Scan(&view.ID, &view.Part, &view.Name, &view.Host, &view.Path, &view.Version, &view.Restapi, &view.Active, &view.Description, &view.Image, &view.Userid, &view.Updated, &view.Favorite)

        if err != nil {
            return nil, err
        }

        views = append(views, view)
    }

    return views, nil
}

func (a *apigatewayRepository) GetMicroserviceApi(id int) (domain.MicroApi, error) {

    view := domain.MicroApi{}

    err := psql.Select(`
            a.id,
            a.part,
            a.org_guid,
            a.name,
            a.host,
            a.path,
            a.version,
            a.rest_api,
            a.active,
            a.description,
            a.image,
            a.updated,
            a.user_id,
            b.micro_id
        `).
        From("micro_api a, micro_api_frontend b").
        Where(sq.Expr("a.id = b.api_id")).
        Where(sq.Eq{"a.id": id}).
        Where(sq.Eq{"a.active": "Y"}).
        RunWith(a.conn).
        QueryRow().
        Scan(&view.ID, &view.Part, &view.OrgGuid, &view.Name, &view.Host, &view.Path, &view.Version, &view.Restapi, &view.Active, &view.Description, &view.Image, &view.Updated, &view.Userid, &view.MicroId)

    if err != nil {
        return view, err
    }

    return view, nil
}

func (a *apigatewayRepository) GetMicroserviceApiRule() (domain.MicroApiRule, error) {

    view := domain.MicroApiRule{}

    err := psql.Select("id, rule, active").
        From("micro_api_rule").
        Where(sq.Eq{"active": "Y"}).
        RunWith(a.conn).
        QueryRow().
        Scan(&view.ID, &view.Rule, &view.Active)

    if err != nil {
        if err == sql.ErrNoRows {
            return view, nil
        }
        return view, err
    }

    return view, nil
}

func (a *apigatewayRepository) ListMicroserviceFrontend(spaces []string) ([]domain.View, error) {

    rows, err := psql.Select(`
            m.id,
            COALESCE(m.url,'') url,
            COALESCE(m.swagger,'') swagger,
            COALESCE(m.description,'') description
        `).
        From("micro_app m").
        Where(sq.Eq{"space_guid": spaces}).
        Where(sq.Eq{"active": "Y"}).
        Where(sq.NotEq{"url": nil}).
        RunWith(a.conn).
        Query()

    if err != nil {
        return nil, err
    }

    defer rows.Close()

    views := []domain.View{}
    for rows.Next() {
        view := domain.View{}


        err := rows.Scan(&view.ID, &view.Url, &view.Swagger, &view.Description)

        if err != nil {
            return nil, err
        }

        views = append(views, view)
    }

    return views, nil
}

func (a *apigatewayRepository) GetMicroserviceNameCheck(name string) (bool, error) {

    var count int

    err := psql.Select("count(id)").
        From("micro_api").
        Where(sq.Eq{"name": name}).
        Where(sq.Eq{"active": "Y"}).
        RunWith(a.conn).
        QueryRow().
        Scan(&count)
    if count > 0 {
        return false, nil
    }

    if err != nil {
        return false, err
    }

    return true, nil
}

func (a *apigatewayRepository) GetMicroservicePathCheck(path string) (bool, error) {

    var count int

    err := psql.Select("count(id)").
        From("micro_api").
        Where(sq.Eq{"path": path}).
        Where(sq.Eq{"active": "Y"}).
        RunWith(a.conn).
        QueryRow().
        Scan(&count)
    if count > 0 {
        return false, nil
    }
    if err != nil {
        return false, err
    }
    return true, nil
}


func (a *apigatewayRepository) SaveMicroserviceApi(view domain.MicroApi) error {

    tx, err := a.conn.Begin()
    if err != nil {
        return err
    }

    defer tx.Rollback()

    // api
    var count int
    err = tx.QueryRow(`
        SELECT count(id)
        FROM micro_api
        WHERE id = $1
        `, view.ID).Scan(&count)
    if err != nil {
        return err
    }

    var id int

    if count == 0 {

        //prop, _ := json.Marshal(r.Properties)

        err = psql.Insert("micro_api").
            Columns("part, name, org_guid, host, path, version, rest_api, description, image, user_id, updated").
            Values(view.Part, view.Name, view.OrgGuid, view.Host, view.Path, view.Version, view.Restapi, view.Description, view.Image, view.Userid, "now()").
            Suffix("RETURNING id").
            RunWith(tx).
            QueryRow().
            Scan(&id)

        if err != nil {
            return err
        }
    } else {
        //prop, _ := json.Marshal(r.Properties)

        _, err = psql.Update("micro_api").
            Set("part", view.Part).
            Set("name", view.Name).
            Set("host", view.Host).
            Set("path", view.Path).
            Set("rest_api", view.Restapi).
            Set("description", view.Description).
            Set("user_id", view.Userid).
            Set("updated", sq.Expr("now()")).
            Where(sq.Eq{"id": view.ID}).
            RunWith(tx).
            Exec()

        if err != nil {
            return err
        }
    }

    // rule
    err = tx.QueryRow(`
        SELECT count(id)
        FROM micro_api_rule
        WHERE active = 'Y'
        `).Scan(&count)
    if err != nil {
        return err
    }
    if count == 0 {

        //prop, _ := json.Marshal(r.Properties)

        _, err = psql.Insert("micro_api_rule").
            Columns("rule").
            Values(view.Rule).
            RunWith(tx).
            Exec()

        if err != nil {
            return err
        }
    } else {
        //prop, _ := json.Marshal(r.Properties)

        _, err = psql.Update("micro_api_rule").
            Set("rule", view.Rule).
            Where(sq.Eq{"active": "Y"}).
            RunWith(tx).
            Exec()

        if err != nil {
            return err
        }
    }

    // frontend
    err = tx.QueryRow(`
        SELECT count(id)
        FROM micro_api_frontend
        WHERE api_id = $1
        `, view.ID).Scan(&count)
    if err != nil {
        return err
    }

    if count == 0 {

        _, err = psql.Insert("micro_api_frontend").
            Columns("api_id, micro_id").
            Values(id, view.MicroId).
            RunWith(tx).
            Exec()

        if err != nil {
            return err
        }
    } else {

        _, err = psql.Update("micro_api_frontend").
            Set("micro_id", view.MicroId).
            Where(sq.Eq{"api_id": view.ID}).
            RunWith(tx).
            Exec()

        if err != nil {
            return err
        }
    }

    err = tx.Commit()
    if err != nil {
        return err
    }

    return nil
}

func (a *apigatewayRepository) ListMicroserviceAppApiByMicroID(id int) ([]domain.MicroApi, error) {

    views := []domain.MicroApi{}

    rows, err := psql.Select(`
            c.micro_id,
            c.api_id,
            c.username,
            b.name,
            b.host,
            b.path,
            b.image
        `).
        From("micro_app a, micro_api b, micro_app_api c").
        Where(sq.Expr("a.id = c.micro_id")).
        Where(sq.Expr("b.id = c.api_id")).
        Where(sq.Eq{"a.id": id}).
        Where(sq.Eq{"a.active": "Y"}).
        Where(sq.Eq{"b.active": "Y"}).
        Where(sq.Eq{"c.active": "Y"}).
        OrderBy("b.name").
        RunWith(a.conn).
        Query()

    if err != nil {
        if err == sql.ErrNoRows {
            return views, nil
        }
        return nil, err
    }

    defer rows.Close()


    for rows.Next() {
        view := domain.MicroApi{}


        err := rows.Scan(&view.MicroId, &view.ID, &view.Username, &view.Name, &view.Host, &view.Path, &view.Image)

        if err != nil {
            return nil, err
        }

        views = append(views, view)
    }

    return views, nil
}

func (a *apigatewayRepository) CreateMicroserviceAppApi(view domain.MicroApi) (bool, error) {

    tx, err := a.conn.Begin()
    if err != nil {
        return false, err
    }

    defer tx.Rollback()

    // api
    var count int

    err = tx.QueryRow(`
        SELECT count(id)
        FROM micro_app_api
        WHERE username = $1
        AND api_id = $2
        AND active = 'Y'
        `, view.Username, view.ID).Scan(&count)
    if err != nil {
        return false, err
    }

    if count != 0 {
        return false, errors.New("Duplicate user")
    }

    err = tx.QueryRow(`
        SELECT count(id)
        FROM micro_app_api
        WHERE micro_id = $1
        AND api_id = $2
        AND active = 'Y'
        `, view.MicroId, view.ID).Scan(&count)
    if err != nil {
        return false, err
    }

    result := false

    if count == 0 {

        sqlresult, err := psql.Insert("micro_app_api").
            Columns("micro_id, api_id, username").
            Values(view.MicroId, view.ID, view.Username).
            RunWith(tx).
            Exec()

        if err != nil {
            return false, err
        }

        rows, err := sqlresult.RowsAffected()
        if err != nil {
            return false, err
        }

        result = (rows == 1)

    }
    err = tx.Commit()
    if err != nil {
        return false, err
    }
    if result != true {
        return result, errors.New("This API already Exists!")
    }

    return result, nil
}


func (a *apigatewayRepository) SaveMicroserviceApiRule(view domain.MicroApi) error {

    tx, err := a.conn.Begin()
    if err != nil {
        return err
    }

    defer tx.Rollback()

    // rule
    var count int

    err = tx.QueryRow(`
        SELECT count(id)
        FROM micro_api_rule
        WHERE active = 'Y'
        `).Scan(&count)
    if err != nil {
        return err
    }
    if count == 0 {

        _, err = psql.Insert("micro_api_rule").
            Columns("rule").
            Values(view.Rule).
            RunWith(tx).
            Exec()

        if err != nil {
            return err
        }
    } else {
        //prop, _ := json.Marshal(r.Properties)

        _, err = psql.Update("micro_api_rule").
            Set("rule", view.Rule).
            Where(sq.Eq{"active": "Y"}).
            RunWith(tx).
            Exec()

        if err != nil {
            return err
        }
    }


    err = tx.Commit()
    if err != nil {
        return err
    }

    return nil
}


func (a *apigatewayRepository) DeleteMicroserviceApi(view domain.MicroApi, backupRule string) error {

    tx, err := a.conn.Begin()
    if err != nil {
        return err
    }

    defer tx.Rollback()

    deletedDate := time.Now().Format("2006-01-02 15:04:05")
    _, err = psql.Update("micro_api").
        Set("active", "N").
        Set("name", view.Name +"__"+ deletedDate).
        Where(sq.Eq{"id": view.ID}).
        RunWith(tx).
        Exec()

    if err != nil {
        return err
    }

    // rule
    var count int

    err = tx.QueryRow(`
        SELECT count(id)
        FROM micro_api_rule
        WHERE active = 'N'
        `).Scan(&count)
    if err != nil {
        return err
    }

    if count == 0 {
        // backcup
        _, err = psql.Insert("micro_api_rule").
            Columns("rule, active").
            Values(backupRule, "N").
            RunWith(tx).
            Exec()

        if err != nil {
            return err
        }

        // new insert
        _, err = psql.Update("micro_api_rule").
            Set("rule", view.Rule).
            Where(sq.Eq{"active": "Y"}).
            RunWith(tx).
            Exec()

        if err != nil {
            return err
        }
    } else {
        // backup
        _, err = psql.Update("micro_api_rule").
            Set("rule", backupRule).
            Where(sq.Eq{"active": "N"}).
            RunWith(tx).
            Exec()

        if err != nil {
            return err
        }

        // new insert
        _, err = psql.Update("micro_api_rule").
            Set("rule", view.Rule).
            Where(sq.Eq{"active": "Y"}).
            RunWith(tx).
            Exec()

        if err != nil {
            return err
        }
    }

    _, err = psql.Update("micro_app_api").
        Set("active", "N").
        Where(sq.Eq{"api_id": view.ID}).
        RunWith(tx).
        Exec()

    if err != nil {
        return err
    }


    err = tx.Commit()
    if err != nil {
        return err
    }

    return nil
}


func (a *apigatewayRepository) DeleteMicroserviceAppApi(id int, microid int, username string) error {

    tx, err := a.conn.Begin()
    if err != nil {
        return err
    }

    defer tx.Rollback()

    //deleted_date := time.Now().Format("2006-01-02 15:04:05")

    _, err = psql.Update("micro_app_api").
        Set("active", "N").
		//Set("username", username+"__"+deleted_date).
        Where(sq.Eq{"api_id": id}).
        Where(sq.Eq{"micro_id": microid}).
        Where(sq.Eq{"username": username}).
        RunWith(tx).
        Exec()

    if err != nil {
        return err
    }


    err = tx.Commit()
    if err != nil {
        return err
    }

    return nil
}

func (v *apigatewayRepository) SaveMicroserviceSwaggerApi(view domain.View) error {

    _, err := psql.Update("micro_app").
        Set("swagger", view.Swagger).
        Where(sq.Eq{"id": view.ID}).
        RunWith(v.conn).
        Exec()

    if err != nil {
        return err
    }


    return nil
}

func (a *apigatewayRepository) GetMicroserviceAppApi(microId int, apiId int) (domain.MicroAppApi, error) {
    appApi := domain.MicroAppApi{}

    err := psql.Select("id, micro_id, api_id, username, active").
        From("micro_app_api").
        Where(sq.And{
            sq.Eq{"micro_id": microId},
            sq.Eq{"api_id": apiId},
            sq.Eq{"active": "Y"}}).
        RunWith(a.conn).
        QueryRow().
        Scan(&appApi.ID, &appApi.MicroId, &appApi.ApiId, &appApi.UserName, &appApi.Active)

    if err != nil {
        if err == sql.ErrNoRows {
            return appApi, nil
        }
        return appApi, err
    }

    return appApi, nil
}


func (a *apigatewayRepository) ListMicroserviceAppApiByApiID(apiId int) ([]domain.MicroApi, error) {
    rows,err := psql.Select(`
            appApi.micro_id,
            api.path,
            appApi.username,
            appApi.active,
            appApi.api_id,
	        api.name,
            service.name
        `).
		From("micro_app_api appApi, micro_api api, micro_app service").
	    Where(sq.Expr("appApi.micro_id = service.id")).
        Where(sq.Expr("appApi.api_id = api.id")).
        Where(sq.Eq{"appApi.api_id": apiId}).
        Where(sq.Eq{"service.active": "Y"}).
        Where(sq.Eq{"appApi.active": "Y"}).
        Where(sq.Eq{"api.active": "Y"}).
		RunWith(a.conn).
		Query()
	/*
Where(sq.And{
	//sq.Eq{"micro_id": microId},
	sq.Eq{"api_id": apiId},
	sq.Eq{"active": "Y"}}).
*/
    if err != nil {
        return nil, err
    }

    defer rows.Close()

    appApis := []domain.MicroApi{}
    for rows.Next() {
        appApi := domain.MicroApi{}

        err := rows.Scan(&appApi.MicroId, &appApi.Path, &appApi.Username, &appApi.Active, &appApi.ID, &appApi.Name, &appApi.MicroName )

        if err != nil {
            return nil, err
        }

        appApis = append(appApis, appApi)
    }

	return appApis, nil
}
