package db

import (
	"crossent/micro/studio/db/lock"
	"crossent/micro/studio/domain"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"time"
)

//var ErrConfigComparisonFailed = errors.New("comparison with existing config failed during save")
//var ErrTeamDisappeared = errors.New("temp disappeared")


type ViewRepository interface {
	ID() int
	ListMicroservice(int, string, []string) ([]domain.View, error)
	GetMicroservice(int) (domain.View, error)
	ListMicroserviceAppApp(int) ([]domain.View, error)
	ListMicroserviceApi(int, string, []string) ([]domain.View, error)
	SaveMicroserviceApi(domain.View, []string) error
	DeleteMicroservice(int, string, bool) error
	ListMicroserviceAppService(int) ([]domain.View, error)
}

type viewRepository struct {
	id          int
	conn        Conn
	lockFactory lock.LockFactory
	name        string
}

func (v *viewRepository) ID() int { return v.id }

func newViewRepository(conn Conn, lockFactory lock.LockFactory) *viewRepository {
	return &viewRepository{
		conn:        conn,
		lockFactory: lockFactory,
	}
}

func (v *viewRepository) ListMicroservice(offset int, name string, spaces []string) ([]domain.View, error) {
	condition := sq.Eq{"1": "1"}
	if name != "" {
		condition = sq.Eq{"name": name}
	}
	fmt.Println(spaces)
	rows, err := psql.Select(`
			m.id,
			m.name,
			m.org_guid,
			m.space_guid,
			m.version,
			m.description,
			m.visible,
			COALESCE(m.status,'') status,
			COALESCE(m.url,'') url
		`).
		From("micro_app m").
		//LeftJoin("teams t ON p.team_id = t.id").
	        //Join("jobs j ON b.job_id = j.id").
		//Where(sq.Eq{"team_id": 1}).
		Where(condition).
		Where(sq.Eq{"active": "Y"}).
		Where(sq.Eq{"space_guid": spaces}).
		OrderBy("id").
		Offset(uint64(offset)).
		Limit(6).
		RunWith(v.conn).
		Query()

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	views := []domain.View{}
	for rows.Next() {
		view := domain.View{}


		err := rows.Scan(&view.ID, &view.Name, &view.OrgGuid, &view.SpaceGuid, &view.Version, &view.Description, &view.Visible, &view.Status, &view.Url)

		if err != nil {
			return nil, err
		}

		err = psql.Select("count(app_guid)").
			From("micro_app_app").
			Where(sq.Eq{"micro_id": view.ID}).
			RunWith(v.conn).
			QueryRow().
			Scan(&view.App)

		if err != nil {
			return nil, err
		}

		views = append(views, view)
	}

	return views, nil
}

func (v *viewRepository) GetMicroservice(id int) (domain.View, error) {
	view := domain.View{}

	err := psql.Select("id, name, org_guid, space_guid, version, description, visible, COALESCE(status, ' '), COALESCE(url,''), COALESCE(swagger,'')").
		From("micro_app").
		Where(sq.Eq{"id": id}).
		Where(sq.Eq{"active": "Y"}).
		RunWith(v.conn).
		QueryRow().
		Scan(&view.ID, &view.Name, &view.OrgGuid, &view.SpaceGuid, &view.Version, &view.Description, &view.Visible, &view.Status, &view.Url, &view.Swagger)

	if err != nil {
		return view, err
	}

	err = psql.Select("count(app_guid)").
		From("micro_app_app").
		Where(sq.Eq{"micro_id": view.ID}).
		RunWith(v.conn).
		QueryRow().
		Scan(&view.App)

	if err != nil {
		return view, err
	}

	return view, nil
}

func (v *viewRepository) ListMicroserviceAppApp(id int) ([]domain.View, error) {
	rows, err := psql.Select(`
			micro_id,
			app_guid,
			essential
		`).
		From("micro_app_app").
		Where(sq.Eq{"micro_id": id}).
		RunWith(v.conn).
		Query()

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	views := []domain.View{}
	for rows.Next() {
		view := domain.View{}


		err := rows.Scan(&view.ID, &view.AppGuid, &view.Essential)

		if err != nil {
			return nil, err
		}

		views = append(views, view)
	}

	return views, nil
}

func (v *viewRepository) ListMicroserviceApi(offset int, name string, spaces []string) ([]domain.View, error) {
	condition := sq.Eq{"1": "1"}
	if name != "" {
		condition = sq.Eq{"name": name}
	}
	fmt.Println(spaces)
	rows, err := psql.Select(`
			m.id,
			m.name,
			m.org_guid,
			m.space_guid,
			m.version,
			m.description,
			m.visible,
			COALESCE(m.status,'') status,
			COALESCE(m.url,'') url,
			COALESCE(m.swagger,'') swagger
		`).
		From("micro_app m").
		Where(condition).
		Where(sq.NotEq{"swagger": nil}).
		Where(sq.Or{sq.Eq{"space_guid": spaces}, sq.Eq{"visible": "public"}}).
		OrderBy("id").
		Offset(uint64(offset)).
		Limit(6).
		RunWith(v.conn).
		Query()

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	views := []domain.View{}
	for rows.Next() {
		view := domain.View{}


		err := rows.Scan(&view.ID, &view.Name, &view.OrgGuid, &view.SpaceGuid, &view.Version, &view.Description, &view.Visible, &view.Status, &view.Url, &view.Swagger)

		if err != nil {
			return nil, err
		}

		views = append(views, view)
	}

	return views, nil
}

func (v *viewRepository) SaveMicroserviceApi(view domain.View, spaces []string) error {

	_, err := psql.Update("micro_app").
		Set("url", view.Url).
		Set("swagger", view.Swagger).
		Where(sq.Eq{"id": view.ID}).
		RunWith(v.conn).
		Exec()

	if err != nil {
		return err
	}


	return nil
}

func (v *viewRepository) DeleteMicroservice(id int, name string, deleteMicroApp bool) error {
	tx, err := v.conn.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = psql.Delete("micro_app_app").
		Where(sq.Eq{
		"micro_id": id,
		}).
		RunWith(tx).
		Exec()

	if err != nil {
		return err
	}

	_, err = psql.Delete("micro_app_service").
		Where(sq.Eq{
		"micro_id": id,
	}).
		RunWith(tx).
		Exec()

	if err != nil {
		return err
	}

	if deleteMicroApp == true {
		_, err = psql.Delete("micro_app").
			Where(sq.Eq{
				"name": name,
			}).
			RunWith(tx).
			Exec()

		if err != nil {
			return err
		}
	} else {
		deletedDate := time.Now().Format("2006-01-02 15:04:05")
		_, err = psql.Update("micro_app").
			Set("name", name+"__"+deletedDate).
			Set("active", "N").
			Where(sq.Eq{"id": id}).
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

func (v *viewRepository) ListMicroserviceAppService(id int) ([]domain.View, error) {
	rows, err := psql.Select(`
			micro_id,
			service_guid
		`).
		From("micro_app_service").
		Where(sq.Eq{"micro_id": id}).
		RunWith(v.conn).
		Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	views := []domain.View{}
	for rows.Next() {
		view := domain.View{}


		err := rows.Scan(&view.ID, &view.ServiceGuid)

		if err != nil {
			return nil, err
		}

		views = append(views, view)
	}

	return views, nil
}